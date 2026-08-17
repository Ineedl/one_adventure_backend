// Package xds 提供基于 etcd 的 Envoy xDS 控制面实现。
package xds

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"net"
	"sort"
	"strings"
	"time"

	observabilitylog "one_adventure_observability_log"
	"one_adventure_servicekit/discovery"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	grpccluster "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	grpcdiscovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	grpcendpoint "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resources "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

var runtimeLogger *observabilitylog.Logger

// xdsLog 同时写入标准输出和 Loki，保证本地调试与集中检索都可用。
type xdsLog struct{}

func (xdsLog) Printf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	stdlog.Print(message)
	if runtimeLogger != nil {
		runtimeLogger.Info(context.Background(), message)
	}
}

var log xdsLog

// envoyNodeID 必须与 Envoy bootstrap 中 node.id 保持一致，SnapshotCache 才能命中对应配置。
const envoyNodeID = "one-adventure"

// Config 描述 xDS 控制面的监听参数和 etcd 数据源。
type Config struct {
	// Endpoints 是 etcd 客户端地址列表。
	Endpoints []string
	// ListenAddress 和 ListenPort 定义 xDS gRPC 服务监听地址。
	ListenAddress string
	ListenPort    int
	// DialTimeout 是连接 etcd 的超时时间。
	DialTimeout time.Duration
	// Services 可选，用于限制只下发指定服务的配置。
	Services []string
}

// Run 启动 xDS gRPC 服务，并持续从 etcd 刷新配置快照直到 ctx 结束。
func Run(ctx context.Context, cfg Config) error {
	runtimeLogger = observabilitylog.New(observabilitylog.Config{Enabled: true, ServiceName: "xds", InstanceID: observabilitylog.NewInstanceID("xds"), LokiURL: "http://127.0.0.1:3100/loki/api/v1/push", Labels: map[string]string{"component": "xds"}})
	defer runtimeLogger.Shutdown(context.Background())
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("etcd endpoints are required")
	}
	if cfg.ListenPort == 0 {
		cfg.ListenPort = 18000
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "0.0.0.0"
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	log.Printf("xds: connecting to etcd endpoints=%v", cfg.Endpoints)
	// clientv3.New 创建 etcd v3 客户端，后续全量读取和 Watch 都复用该连接。
	ec, err := clientv3.New(clientv3.Config{Endpoints: cfg.Endpoints, DialTimeout: cfg.DialTimeout})
	if err != nil {
		return err
	}
	defer ec.Close()
	// Status 主动探测第一个 etcd endpoint，尽早暴露连接故障。
	if _, err := ec.Status(ctx, cfg.Endpoints[0]); err != nil {
		log.Printf("xds: etcd connection check failed: %v", err)
	} else {
		log.Printf("xds: etcd connected endpoint=%s", cfg.Endpoints[0])
	}
	// SnapshotCache 保存各 Envoy node 对应的最新 CDS/EDS 一致性快照。
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	// CallbackFuncs 接收 Envoy ADS 流的建立、请求、响应和关闭事件。
	callbacks := server.CallbackFuncs{StreamOpenFunc: func(_ context.Context, id int64, typ string) error {
		log.Printf("xds: envoy stream opened id=%d type=%s", id, typ)
		return nil
	}, StreamClosedFunc: func(id int64, node *core.Node) { log.Printf("xds: envoy stream closed id=%d node=%v", id, node) }, StreamRequestFunc: func(id int64, req *grpcdiscovery.DiscoveryRequest) error {
		log.Printf("xds: envoy request id=%d type=%s version=%s", id, req.TypeUrl, req.VersionInfo)
		return nil
	}, StreamResponseFunc: func(_ context.Context, id int64, req *grpcdiscovery.DiscoveryRequest, _ *grpcdiscovery.DiscoveryResponse) {
		log.Printf("xds: response sent id=%d type=%s nonce=%s", id, req.TypeUrl, req.ResponseNonce)
	}}
	// go-control-plane Server 根据 Envoy 请求从 SnapshotCache 生成 xDS 响应。
	srv := server.NewServer(ctx, cache, callbacks)
	// xDS 使用 gRPC 双向流传输动态配置。
	g, err := grpc.NewServer(), error(nil)
	if err != nil {
		return err
	}
	// 注册 ADS、CDS 和 EDS gRPC 服务，使 Envoy 能订阅集群与 endpoint。
	grpcdiscovery.RegisterAggregatedDiscoveryServiceServer(g, srv)
	grpccluster.RegisterClusterDiscoveryServiceServer(g, srv)
	grpcendpoint.RegisterEndpointDiscoveryServiceServer(g, srv)
	// net.Listen 创建对 Envoy 暴露的 xDS TCP 监听器。
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort))
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("xds: listening for Envoy xDS on %s", ln.Addr())
	go watch(ctx, ec, cache, cfg.Services)
	// GracefulStop 等待已有 xDS 流退出，避免进程关闭时直接中断 Envoy。
	go func() { <-ctx.Done(); g.GracefulStop() }()
	return g.Serve(ln)
}

func watch(ctx context.Context, ec *clientv3.Client, c cache.SnapshotCache, services []string) {
	// 先生成初始快照，再在 etcd 发生变更时重新生成并推送。
	prefixes := services
	if len(prefixes) == 0 {
		prefixes = []string{"*"}
	}
	for {
		snap := buildSnapshot(ctx, ec, prefixes)
		if snap != nil {
			// SetSnapshot 发布新版本，正在订阅的 Envoy 会立即收到增量变化。
			c.SetSnapshot(ctx, envoyNodeID, snap)
			log.Printf("xds: snapshot published services=%s", snapshotServices(prefixes))
		}
		if len(services) > 0 {
			for _, s := range services {
				// etcd Watch 以服务前缀监听实例新增、更新和租约删除事件。
				w := ec.Watch(ctx, discovery.RootPrefix+"/"+strings.Trim(s, "/")+"/", clientv3.WithPrefix())
				for r := range w {
					if r.Err() == nil {
						log.Printf("xds: etcd event received service=%s revision=%d", s, r.Header.Revision)
						snap = buildSnapshot(ctx, ec, services)
						if snap != nil {
							c.SetSnapshot(ctx, envoyNodeID, snap)
						}
					}
				}
			}
		} else {
			// 未限定服务时监听注册根目录下的所有实例变化。
			w := ec.Watch(ctx, discovery.RootPrefix+"/", clientv3.WithPrefix())
			for r := range w {
				if r.Err() != nil {
					log.Printf("xds: etcd watch failed: %v", r.Err())
					break
				}
				log.Printf("xds: etcd event received revision=%d", r.Header.Revision)
				snap = buildSnapshot(ctx, ec, prefixes)
				if snap != nil {
					c.SetSnapshot(ctx, envoyNodeID, snap)
					log.Printf("xds: snapshot published services=%s", snapshotServices(prefixes))
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func buildSnapshot(ctx context.Context, ec *clientv3.Client, services []string) *cache.Snapshot {
	// 注册包负责约定键值格式，xDS 只负责将实例转换为 Envoy 的 CDS/EDS 资源。
	// WithPrefix 让 etcd Get 一次读取注册根路径下的全部实例。x
	resp, err := ec.Get(ctx, discovery.RootPrefix+"/", clientv3.WithPrefix())
	if err != nil {
		return nil
	}
	eps := map[string]map[localityKey][]*endpoint.LbEndpoint{}
	names := map[string]bool{}
	for _, kv := range resp.Kvs {
		parts := strings.Split(strings.TrimPrefix(string(kv.Key), discovery.RootPrefix+"/"), "/")
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if len(services) > 0 && services[0] != "*" {
			ok := false
			for _, s := range services {
				if s == name {
					ok = true
				}
			}
			if !ok {
				continue
			}
		}
		var in discovery.Instance
		if json.Unmarshal(kv.Value, &in) != nil {
			continue
		}
		address := resolveEndpointAddress(in.Address)
		if address == "" {
			log.Printf("xds: skip endpoint service=%s address=%s: cannot resolve address", name, in.Address)
			continue
		}
		key := localityKey{region: strings.TrimSpace(in.Region), zone: strings.TrimSpace(in.Zone), subZone: strings.TrimSpace(in.SubZone)}
		if eps[name] == nil {
			eps[name] = map[localityKey][]*endpoint.LbEndpoint{}
		}
		eps[name][key] = append(eps[name][key], &endpoint.LbEndpoint{HostIdentifier: &endpoint.LbEndpoint_Endpoint{Endpoint: &endpoint.Endpoint{Address: &core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{Address: address, PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(port(in.GRPCPort))}}}}}}})
		names[name] = true
	}
	clusters := []types.Resource{}
	endpoints := []types.Resource{}
	for n := range names {
		// Cluster 使用 EDS 发现实例，并启用 HTTP/2 以承载微服务 gRPC 请求。
		clusters = append(clusters, &cluster.Cluster{Name: n, ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS}, Http2ProtocolOptions: &core.Http2ProtocolOptions{}, EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{ServiceName: n, EdsConfig: &core.ConfigSource{ConfigSourceSpecifier: &core.ConfigSource_Ads{Ads: &core.AggregatedConfigSource{}}}}})
		endpoints = append(endpoints, &endpoint.ClusterLoadAssignment{ClusterName: n, Endpoints: localityLbEndpoints(eps[n])})
	}
	// NewSnapshot 使用 etcd revision 作为版本号，保证 CDS 与 EDS 同版本发布。
	s, _ := cache.NewSnapshot(fmt.Sprint(resp.Header.Revision), map[resources.Type][]types.Resource{resources.EndpointType: endpoints, resources.ClusterType: clusters})
	return s
}

type localityKey struct {
	region  string
	zone    string
	subZone string
}

func localityLbEndpoints(groups map[localityKey][]*endpoint.LbEndpoint) []*endpoint.LocalityLbEndpoints {
	keys := make([]localityKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].region != keys[j].region {
			return keys[i].region < keys[j].region
		}
		if keys[i].zone != keys[j].zone {
			return keys[i].zone < keys[j].zone
		}
		return keys[i].subZone < keys[j].subZone
	})

	result := make([]*endpoint.LocalityLbEndpoints, 0, len(keys))
	for _, key := range keys {
		group := &endpoint.LocalityLbEndpoints{LbEndpoints: groups[key]}
		if key != (localityKey{}) {
			group.Locality = &core.Locality{Region: key.region, Zone: key.zone, SubZone: key.subZone}
		}
		result = append(result, group)
	}
	return result
}

// resolveEndpointAddress 将注册信息中的主机名解析为 Envoy EDS 要求的 IP 地址。
func resolveEndpointAddress(address string) string {
	if ip := net.ParseIP(strings.TrimSpace(address)); ip != nil {
		return ip.String()
	}
	ips, err := net.LookupIP(strings.TrimSpace(address))
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String()
		}
	}
	return ""
}

// snapshotServices 将日志中的服务范围格式化为易读文本。
func snapshotServices(services []string) string {
	if len(services) == 1 && services[0] == "*" {
		return "all"
	}
	return strings.Join(services, ",")
}

// port 将服务注册中字符串形式的端口转换为 Envoy 使用的数字端口。
func port(s string) int { var p int; fmt.Sscan(s, &p); return p }
