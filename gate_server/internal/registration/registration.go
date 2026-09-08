package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	servermanagerpb "one_adventure_rpc/proto/server_manager"
	"one_adventure_servicekit/api-contract/gateway_server_discovery"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	EtcdEndpoints []string
	DialTimeout   time.Duration
	EnvoyAddress  string
	LeaseTTL      int64
	Address       string
	WSPort        int
	InstanceID    string
}

type Info struct {
	Address    string `json:"address"`
	WSPort     int    `json:"ws_port"`
	InstanceID string `json:"instance_id"`
}
type Lease struct {
	client *clientv3.Client
	id     clientv3.LeaseID
	key    string
}

func FetchServerInfo(ctx context.Context, envoyAddress, name string) (*servermanagerpb.ServerInfo, error) {
	conn, err := grpc.NewClient(envoyAddress, grpc.WithAuthority("server_manager"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// The server has already been claimed by this gate instance, so it is no
	// longer part of the type=1 (unoccupied) result. Query all servers here.
	resp, err := servermanagerpb.NewServerManagerServiceClient(conn).ServerInfoGet(ctx, &servermanagerpb.ServerInfoGetReq{Type: 0})
	if err != nil {
		return nil, err
	}
	for _, server := range resp.GetServers() {
		if server.GetName() == name {
			return server, nil
		}
	}
	return nil, fmt.Errorf("server %q not found in server manager response", name)
}

func Acquire(ctx context.Context, cfg Config) (*Lease, string, error) {
	if len(cfg.EtcdEndpoints) == 0 || cfg.EnvoyAddress == "" || cfg.Address == "" || cfg.WSPort < 1 || cfg.LeaseTTL <= 0 {
		return nil, "", errors.New("invalid gate registration config")
	}
	conn, err := grpc.NewClient(cfg.EnvoyAddress, grpc.WithAuthority("server_manager"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, "", fmt.Errorf("connect server manager: %w", err)
	}
	defer conn.Close()
	client := servermanagerpb.NewServerManagerServiceClient(conn)
	etcd, err := clientv3.New(clientv3.Config{Endpoints: cfg.EtcdEndpoints, DialTimeout: cfg.DialTimeout})
	if err != nil {
		return nil, "", fmt.Errorf("connect etcd: %w", err)
	}
	for {
		resp, e := client.ServerInfoGet(ctx, &servermanagerpb.ServerInfoGetReq{Type: 1})
		if e != nil {
			etcd.Close()
			return nil, "", fmt.Errorf("get unplayed servers: %w", e)
		}
		if len(resp.GetServers()) == 0 {
			etcd.Close()
			return nil, "", nil
		}
		claimed := false
		for _, server := range resp.GetServers() {
			if server.GetName() == "" {
				continue
			}
			grant, e := etcd.Grant(ctx, cfg.LeaseTTL)
			if e != nil {
				etcd.Close()
				return nil, "", e
			}
			key := gateway_server_discovery.ServerKey(server.GetId())
			value, _ := json.Marshal(Info{Address: cfg.Address, WSPort: cfg.WSPort, InstanceID: cfg.InstanceID})
			txn, e := etcd.Txn(ctx).If(clientv3.Compare(clientv3.Version(key), "=", 0)).Then(clientv3.OpPut(key, string(value), clientv3.WithLease(grant.ID))).Commit()
			if e != nil {
				_, _ = etcd.Revoke(ctx, grant.ID)
				etcd.Close()
				return nil, "", e
			}
			if txn.Succeeded {
				return &Lease{client: etcd, id: grant.ID, key: key}, server.GetName(), nil
			}
			_, _ = etcd.Revoke(ctx, grant.ID)
			claimed = true
		}
		if !claimed {
			etcd.Close()
			return nil, "", nil
		}
	}
}

func (l *Lease) KeepAlive(ctx context.Context) error {
	ch, err := l.client.KeepAlive(ctx, l.id)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r, ok := <-ch:
			if !ok || r == nil || r.TTL <= 0 {
				return errors.New("gate server lease expired")
			}
		}
	}
}
func (l *Lease) Release(ctx context.Context) error {
	defer l.client.Close()
	_, err := l.client.Revoke(ctx, l.id)
	return err
}

func LocalAddress() string {
	if v := getenv("GATE_SERVER_ADDRESS"); v != "" {
		return v
	}
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		if i.Flags&net.FlagLoopback == 0 {
			as, _ := i.Addrs()
			for _, a := range as {
				h, _, e := net.ParseCIDR(a.String())
				if e == nil && h.IsPrivate() {
					return h.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

var getenv = os.Getenv
