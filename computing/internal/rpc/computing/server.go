package computing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/grpc"
	computingpb "one_adventure_rpc/proto/computing"
	"one_adventure_servicekit/registration"
)

var ErrServerStarted = errors.New("computing rpc server is already started")

// Server owns the computing gRPC server and its TCP listener.
type Server struct {
	port       int
	grpcServer *grpc.Server
	registrationManager *registration.Manager

	mu               sync.RWMutex
	listener         net.Listener
	registrationStop context.CancelFunc
	registrationDone chan struct{}
}

// New initializes the computing RPC server from application configuration.
func New(ctx context.Context) (*Server, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	manager, err := registration.New(cfg.registrationConfig())
	if err != nil {
		return nil, fmt.Errorf("create registration manager: %w", err)
	}
	return newServer(cfg, newComputingService(), manager), nil
}

func newServer(cfg Config, service computingpb.ComputingServiceServer, manager *registration.Manager) *Server {
	grpcServer := grpc.NewServer()
	computingpb.RegisterComputingServiceServer(grpcServer, service)
	if manager != nil {
		manager.RegisterPingService(grpcServer)
	}
	return &Server{
		port:                cfg.Port,
		grpcServer:          grpcServer,
		registrationManager: manager,
	}
}

// Start listens on the configured port and serves in the background.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return ErrServerStarted
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("listen computing rpc on port %d: %w", s.port, err)
	}
	s.listener = listener
	var registrationCtx context.Context
	if s.registrationManager != nil {
		var cancel context.CancelFunc
		registrationCtx, cancel = context.WithCancel(context.Background())
		s.registrationStop = cancel
		s.registrationDone = make(chan struct{})
	}

	go func() {
		if serveErr := s.grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			g.Log().Errorf(context.Background(), "computing rpc server stopped unexpectedly: %v", serveErr)
		}
	}()
	if s.registrationManager != nil {
		registrationDone := s.registrationDone
		go func(done chan struct{}) {
			defer close(done)
			if runErr := s.registrationManager.Run(registrationCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
				g.Log().Errorf(context.Background(), "computing registration manager stopped unexpectedly: %v", runErr)
			}
		}(registrationDone)
	}
	g.Log().Infof(context.Background(), "computing rpc server listening on port %d", s.port)
	return nil
}

// Address returns the listener address after Start has succeeded.
func (s *Server) Address() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// Shutdown gracefully stops the server. When ctx expires, active RPCs are
// canceled and ctx.Err is returned.
func (s *Server) Shutdown(ctx context.Context) error {
	s.stopRegistration()
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		<-done
		return ctx.Err()
	}
}

// Stop immediately stops the server.
func (s *Server) Stop() {
	s.stopRegistration()
	s.grpcServer.Stop()
}

func (s *Server) stopRegistration() {
	s.mu.Lock()
	cancel := s.registrationStop
	done := s.registrationDone
	s.registrationStop = nil
	s.registrationDone = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
