package user

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/grpc"
	userpb "one_adventure_rpc/proto/user"
	"one_adventure_servicekit/discovery"
	appconfig "user/internal/config"
)

var ErrServerStarted = errors.New("user rpc server is already started")

type Server struct {
	port             int
	grpcServer       *grpc.Server
	registrar        *discovery.Registrar
	discoverer       *discovery.Discoverer
	watchServices    []string
	mu               sync.RWMutex
	listener         net.Listener
	registrationStop context.CancelFunc
	registrationDone chan struct{}
}

func New(ctx context.Context, jwtConfig appconfig.JWTConfig) (*Server, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	registrar, err := discovery.NewRegistrar(cfg.discoveryConfig(), cfg.registration())
	if err != nil {
		return nil, fmt.Errorf("create service registrar: %w", err)
	}
	discoverer, err := discovery.NewDiscoverer(cfg.discoveryConfig())
	if err != nil {
		return nil, fmt.Errorf("create service discovery: %w", err)
	}
	service, err := newDefaultUserService(jwtConfig)
	if err != nil {
		return nil, fmt.Errorf("create user service: %w", err)
	}
	return newServer(cfg, service, registrar, discoverer), nil
}

func newServer(cfg Config, service userpb.UserServiceServer, registrar *discovery.Registrar, discoverer *discovery.Discoverer) *Server {
	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, service)
	return &Server{port: cfg.Port, grpcServer: grpcServer, registrar: registrar, discoverer: discoverer, watchServices: cfg.Etcd.WatchServices}
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return ErrServerStarted
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("listen user rpc on port %d: %w", s.port, err)
	}
	s.listener = listener
	var registrationCtx context.Context
	if s.registrar != nil || s.discoverer != nil {
		var cancel context.CancelFunc
		registrationCtx, cancel = context.WithCancel(context.Background())
		s.registrationStop, s.registrationDone = cancel, make(chan struct{})
	}
	go func() {
		if serveErr := s.grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			g.Log().Errorf(context.Background(), "user rpc server stopped unexpectedly: %v", serveErr)
		}
	}()
	if s.registrar != nil {
		done := s.registrationDone
		go func() {
			defer close(done)
			if runErr := s.registrar.Run(registrationCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
				g.Log().Errorf(context.Background(), "user service registrar stopped unexpectedly: %v", runErr)
			}
		}()
		if s.discoverer != nil {
			go func() { _ = s.discoverer.Run(registrationCtx, s.watchServices) }()
		}
	}
	g.Log().Infof(context.Background(), "user rpc server listening on port %d", s.port)
	return nil
}

func (s *Server) Address() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.stopRegistration()
	done := make(chan struct{})
	go func() { s.grpcServer.GracefulStop(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		<-done
		return ctx.Err()
	}
}

func (s *Server) Stop() { s.stopRegistration(); s.grpcServer.Stop() }

func (s *Server) stopRegistration() {
	s.mu.Lock()
	cancel, done := s.registrationStop, s.registrationDone
	s.registrationStop, s.registrationDone = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	if s.discoverer != nil {
		s.discoverer.Close()
	}
}
