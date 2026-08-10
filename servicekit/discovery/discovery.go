// Package discovery provides etcd-backed service registration and gRPC discovery.
package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

const RootPrefix = "/one_adventure"

var ErrServiceUnavailable = errors.New("service is unavailable")

// Instance is the JSON value stored at /one_adventure/<server_name>/<instance_id>.
type Instance struct {
	Address  string `json:"address"`
	GRPCPort string `json:"grpc_port"`
	HTTPPort string `json:"http_port"`
}

func (i Instance) grpcAddress() (string, error) {
	if strings.TrimSpace(i.Address) == "" || strings.TrimSpace(i.GRPCPort) == "" {
		return "", errors.New("address and grpc_port are required")
	}
	port, err := strconv.Atoi(i.GRPCPort)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid grpc_port %q", i.GRPCPort)
	}
	return net.JoinHostPort(strings.TrimSpace(i.Address), i.GRPCPort), nil
}

type Config struct {
	Endpoints   []string
	DialTimeout time.Duration
	DebugLog    func(message string, args ...any)
}

func (c Config) Validate() error { return c.validate() }

func (c Config) validate() error {
	if len(c.Endpoints) == 0 {
		return errors.New("etcd endpoints are required")
	}
	for _, endpoint := range c.Endpoints {
		if strings.TrimSpace(endpoint) == "" {
			return errors.New("etcd endpoint must not be empty")
		}
	}
	return nil
}

func newClient(config Config) (*clientv3.Client, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	return clientv3.New(clientv3.Config{Endpoints: config.Endpoints, DialTimeout: config.DialTimeout})
}

type Registration struct {
	ServerName string
	InstanceID string
	Instance   Instance
	LeaseTTL   int64
}

func (r Registration) Validate() error { _, err := r.key(); return err }

// WithDefaults fills fields that may be omitted from configuration. The
// generated ID is stable for a machine and gRPC port, while remaining opaque.
func (r Registration) WithDefaults() Registration {
	if strings.TrimSpace(r.InstanceID) == "" {
		r.InstanceID = DefaultInstanceID(r.ServerName, r.Instance.GRPCPort)
	}
	return r
}

func DefaultInstanceID(serverName, grpcPort string) string {
	parts := []string{strings.ToLower(strings.TrimSpace(serverName)), strings.TrimSpace(grpcPort)}
	if hostname, err := os.Hostname(); err == nil {
		parts = append(parts, hostname)
	}
	if interfaces, err := net.Interfaces(); err == nil {
		addresses := make([]string, 0, len(interfaces))
		for _, networkInterface := range interfaces {
			if networkInterface.Flags&net.FlagLoopback == 0 && len(networkInterface.HardwareAddr) > 0 {
				addresses = append(addresses, networkInterface.HardwareAddr.String())
			}
		}
		sort.Strings(addresses)
		parts = append(parts, addresses...)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return strings.ToLower(strings.TrimSpace(serverName)) + "_" + hex.EncodeToString(sum[:6])
}

func (r Registration) key() (string, error) {
	r = r.WithDefaults()
	name := strings.ToLower(strings.TrimSpace(r.ServerName))
	id := strings.TrimSpace(r.InstanceID)
	if name == "" || strings.Contains(name, "/") {
		return "", errors.New("server_name is required and cannot contain '/'")
	}
	if id == "" || strings.Contains(id, "/") {
		return "", errors.New("instance_id is required and cannot contain '/'")
	}
	if _, err := r.Instance.grpcAddress(); err != nil {
		return "", err
	}
	if r.LeaseTTL <= 0 {
		return "", errors.New("lease ttl must be greater than zero")
	}
	return RootPrefix + "/" + name + "/" + id, nil
}

// Registrar maintains an etcd lease for the local service. It retries until ctx ends.
type Registrar struct {
	client       *clientv3.Client
	registration Registration
	debugLog     func(message string, args ...any)
}

func NewRegistrar(config Config, registration Registration) (*Registrar, error) {
	registration = registration.WithDefaults()
	if _, err := registration.key(); err != nil {
		return nil, fmt.Errorf("validate registration: %w", err)
	}
	client, err := newClient(config)
	if err != nil {
		return nil, fmt.Errorf("create etcd client: %w", err)
	}
	return &Registrar{client: client, registration: registration, debugLog: config.DebugLog}, nil
}

func (r *Registrar) Run(ctx context.Context) error {
	defer r.client.Close()
	key, _ := r.registration.key()
	value, _ := json.Marshal(r.registration.Instance)
	for ctx.Err() == nil {
		lease, err := r.client.Grant(ctx, r.registration.LeaseTTL)
		if err == nil {
			_, err = r.client.Put(ctx, key, string(value), clientv3.WithLease(lease.ID))
		}
		if err == nil {
			r.debug("etcd connection established", "operation", "register")
			r.debug("service registered in etcd", "key", key, "server_name", r.registration.ServerName, "instance_id", r.registration.InstanceID, "lease_id", lease.ID)
			keepAlive, keepAliveErr := r.client.KeepAlive(ctx, lease.ID)
			if keepAliveErr == nil {
				for range keepAlive {
					if ctx.Err() != nil {
						return ctx.Err()
					}
				}
				r.debug("etcd service lease keepalive ended", "key", key, "lease_id", lease.ID)
			} else {
				r.debug("etcd lease keepalive failed", "key", key, "lease_id", lease.ID, "error", keepAliveErr)
			}
		}
		if err != nil && ctx.Err() == nil {
			r.debug("service registration attempt failed", "key", key, "error", err)
		}
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return ctx.Err()
}

func (r *Registrar) debug(message string, args ...any) {
	if r.debugLog != nil {
		r.debugLog(message, args...)
	}
}

type Discoverer struct {
	client *clientv3.Client

	mu        sync.RWMutex
	instances map[string]map[string]Instance
	clients   map[string]*serviceConnection
	debugLog  func(message string, args ...any)
}

type serviceConnection struct {
	connection *grpc.ClientConn
	resolver   *manual.Resolver
}

func NewDiscoverer(config Config) (*Discoverer, error) {
	client, err := newClient(config)
	if err != nil {
		return nil, fmt.Errorf("create etcd client: %w", err)
	}
	return &Discoverer{client: client, instances: make(map[string]map[string]Instance), clients: make(map[string]*serviceConnection), debugLog: config.DebugLog}, nil
}

// Run first loads each configured service prefix, then watches from its read
// revision. An empty serviceNames list watches nothing.
func (d *Discoverer) Run(ctx context.Context, serviceNames []string) error {
	return d.run(ctx, serviceNames, false, nil)
}

// RunReady is Run with a readiness notification sent after the initial etcd
// snapshot has been loaded and watches have been installed.
func (d *Discoverer) RunReady(ctx context.Context, serviceNames []string, ready chan<- error) error {
	return d.run(ctx, serviceNames, false, ready)
}

// RunAllReady watches every service below RootPrefix. It is intended for the gateway.
func (d *Discoverer) RunAllReady(ctx context.Context, ready chan<- error) error {
	return d.run(ctx, nil, true, ready)
}

func (d *Discoverer) run(ctx context.Context, serviceNames []string, watchAll bool, ready chan<- error) error {
	prefixes := watchPrefixes(serviceNames, watchAll)
	var group sync.WaitGroup
	for _, prefix := range prefixes {
		response, err := d.client.Get(ctx, prefix, clientv3.WithPrefix())
		if err != nil {
			if ready != nil {
				ready <- err
			}
			d.Close()
			return fmt.Errorf("get etcd services under %s: %w", prefix, err)
		}
		d.debug("etcd connection established", "prefix", prefix, "revision", response.Header.Revision)
		for _, kv := range response.Kvs {
			d.applyPut(string(kv.Key), kv.Value)
		}
		revision := response.Header.Revision + 1
		group.Add(1)
		go func(prefix string, revision int64) {
			defer group.Done()
			for watchResponse := range d.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(revision)) {
				if watchResponse.Err() != nil {
					continue
				}
				for _, event := range watchResponse.Events {
					if event.Type == clientv3.EventTypeDelete {
						d.debug("service instance deleted from etcd", "key", string(event.Kv.Key), "revision", event.Kv.ModRevision)
						d.applyDelete(string(event.Kv.Key))
					} else {
						d.debug("service instance event received from etcd", "event", event.Type.String(), "key", string(event.Kv.Key), "revision", event.Kv.ModRevision)
						d.applyPut(string(event.Kv.Key), event.Kv.Value)
					}
				}
			}
		}(prefix, revision)
	}
	if ready != nil {
		ready <- nil
	}
	<-ctx.Done()
	group.Wait()
	d.Close()
	return ctx.Err()
}

func watchPrefixes(names []string, watchAll bool) []string {
	if watchAll {
		return []string{RootPrefix + "/"}
	}
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" && !strings.Contains(name, "/") {
			seen[RootPrefix+"/"+name+"/"] = struct{}{}
		}
	}
	prefixes := make([]string, 0, len(seen))
	for prefix := range seen {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)
	return prefixes
}

func splitKey(key string) (string, string, bool) {
	parts := strings.Split(strings.TrimPrefix(key, RootPrefix+"/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (d *Discoverer) applyPut(key string, value []byte) {
	service, id, ok := splitKey(key)
	if !ok {
		return
	}
	var instance Instance
	if err := json.Unmarshal(value, &instance); err != nil {
		return
	}
	if _, err := instance.grpcAddress(); err != nil {
		return
	}
	d.mu.Lock()
	if d.instances[service] == nil {
		d.instances[service] = make(map[string]Instance)
	}
	d.instances[service][id] = instance
	d.updateConnectionLocked(service)
	d.mu.Unlock()
	d.debug("service instance added", "service_name", service, "instance_id", id, "address", instance.Address, "grpc_port", instance.GRPCPort)
}

func (d *Discoverer) applyDelete(key string) {
	service, id, ok := splitKey(key)
	if !ok {
		return
	}
	d.mu.Lock()
	if instances := d.instances[service]; instances != nil {
		delete(instances, id)
		if len(instances) == 0 {
			delete(d.instances, service)
		}
	}
	d.updateConnectionLocked(service)
	d.mu.Unlock()
	d.debug("service instance removed", "service_name", service, "instance_id", id)
}

func (d *Discoverer) debug(message string, args ...any) {
	if d.debugLog != nil {
		d.debugLog(message, args...)
	}
}

// Connection returns a gRPC connection whose resolver receives all currently
// registered instances, enabling round_robin client-side load balancing.
func (d *Discoverer) Connection(serviceName string) (grpc.ClientConnInterface, error) {
	name := strings.ToLower(strings.TrimSpace(serviceName))
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.instances[name]) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrServiceUnavailable, name)
	}
	if connection := d.clients[name]; connection != nil {
		return connection.connection, nil
	}
	scheme := "one-adventure-" + strings.ReplaceAll(name, "_", "-")
	manualResolver := manual.NewBuilderWithScheme(scheme)
	connection, err := grpc.NewClient(scheme+":///"+name, grpc.WithResolvers(manualResolver), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`))
	if err != nil {
		return nil, err
	}
	d.clients[name] = &serviceConnection{connection: connection, resolver: manualResolver}
	d.updateConnectionLocked(name)
	return connection, nil
}

func (d *Discoverer) updateConnectionLocked(service string) {
	connection := d.clients[service]
	if connection == nil {
		return
	}
	addresses := make([]resolver.Address, 0, len(d.instances[service]))
	for _, instance := range d.instances[service] {
		if address, err := instance.grpcAddress(); err == nil {
			addresses = append(addresses, resolver.Address{Addr: address})
		}
	}
	sort.Slice(addresses, func(i, j int) bool { return addresses[i].Addr < addresses[j].Addr })
	connection.resolver.UpdateState(resolver.State{Addresses: addresses})
}

func (d *Discoverer) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, connection := range d.clients {
		_ = connection.connection.Close()
	}
	d.clients = make(map[string]*serviceConnection)
	if d.client != nil {
		_ = d.client.Close()
		d.client = nil
	}
}
