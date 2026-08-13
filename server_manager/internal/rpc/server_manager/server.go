package server_manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/grpc"
	tracekit "one_adventure_observability_trace/trace"
	servermanagerpb "one_adventure_rpc/proto/server_manager"
	"one_adventure_servicekit/discovery"
)

var ErrServerStarted = errors.New("server manager rpc server is already started")

type Server struct {
	port       int
	grpcServer *grpc.Server
	registrar  *discovery.Registrar

	mu               sync.RWMutex
	listener         net.Listener
	registrationStop context.CancelFunc
	registrationDone chan struct{}
}

func New(ctx context.Context) (*Server, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	registrar, err := discovery.NewRegistrar(config.discoveryConfig(), config.registration())
	if err != nil {
		return nil, fmt.Errorf("create server manager registrar: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(tracekit.UnaryServerInterceptor), grpc.StreamInterceptor(tracekit.StreamServerInterceptor))
	servermanagerpb.RegisterServerManagerServiceServer(grpcServer, newServerManagerService())
	return &Server{port: config.Port, grpcServer: grpcServer, registrar: registrar}, nil
}

// InstanceID 返回当前服务注册到 etcd 使用的实例唯一标识。
func (s *Server) InstanceID() string { return s.registrar.InstanceID() }

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return ErrServerStarted
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("listen server manager rpc on port %d: %w", s.port, err)
	}
	s.listener = listener
	registrationCtx, cancel := context.WithCancel(context.Background())
	s.registrationStop, s.registrationDone = cancel, make(chan struct{})
	go func() {
		if serveErr := s.grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			g.Log().Errorf(context.Background(), "server manager rpc server stopped unexpectedly: %v", serveErr)
		}
	}()
	done := s.registrationDone
	go func() {
		defer close(done)
		if runErr := s.registrar.Run(registrationCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			g.Log().Errorf(context.Background(), "server manager registrar stopped unexpectedly: %v", runErr)
		}
	}()
	g.Log().Infof(context.Background(), "server manager rpc server listening on port %d", s.port)
	return nil
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
}
