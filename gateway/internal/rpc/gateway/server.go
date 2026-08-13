package gateway

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/grpc"
	"one_adventure_gateway/internal/service"
	"one_adventure_servicekit/discovery"
)

var ErrServerStarted = errors.New("gateway discovery is already started")

// Server owns gateway service discovery. Gateway no longer exposes a service
// registration RPC; it resolves all backends directly from etcd.
type Server struct {
	discoverer *discovery.Discoverer
	instanceID string
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
}

func New(ctx context.Context) (*Server, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	discoverer, err := discovery.NewDiscoverer(cfg.discoveryConfig())
	if err != nil {
		return nil, fmt.Errorf("create service discovery: %w", err)
	}
	instanceID := cfg.Etcd.InstanceID
	if instanceID == "" {
		instanceID = discovery.DefaultInstanceID(cfg.Etcd.ServerName, fmt.Sprint(cfg.Port))
	}
	return &Server{discoverer: discoverer, instanceID: instanceID}, nil
}

// InstanceID 返回 gateway 当前实例的唯一标识。
func (s *Server) InstanceID() string { return s.instanceID }

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return ErrServerStarted
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.done = cancel, make(chan struct{})
	ready := make(chan error, 1)
	go func() {
		defer close(s.done)
		if err := s.discoverer.RunAllReady(ctx, ready); err != nil && !errors.Is(err, context.Canceled) {
			g.Log().Errorf(context.Background(), "gateway service discovery stopped: %v", err)
		}
	}()
	if err := <-ready; err != nil {
		cancel()
		return fmt.Errorf("initialize gateway discovery: %w", err)
	}
	return nil
}

func (s *Server) ResolveService(serviceName string) (grpc.ClientConnInterface, error) {
	connection, err := s.discoverer.Connection(serviceName)
	if errors.Is(err, discovery.ErrServiceUnavailable) {
		return nil, fmt.Errorf("%w: %s", service.ErrServiceUnavailable, serviceName)
	}
	return connection, err
}

func (s *Server) Shutdown(ctx context.Context) error { return s.stop(ctx) }
func (s *Server) Stop()                              { _ = s.stop(context.Background()) }

func (s *Server) stop(ctx context.Context) error {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		s.discoverer.Close()
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.discoverer.Close()
		return ctx.Err()
	}
}
