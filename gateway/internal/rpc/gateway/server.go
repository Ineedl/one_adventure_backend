package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"
	"google.golang.org/grpc"
	gatewaypb "one_adventure_rpc/proto/gateway"
)

var ErrServerStarted = errors.New("gateway rpc server is already started")

// Server owns the gateway gRPC server and its TCP listener.
type Server struct {
	port       int
	grpcServer *grpc.Server

	mu       sync.RWMutex
	listener net.Listener
}

// New initializes the gateway RPC server from application configuration and
// registers GatewayService.
func New(ctx context.Context) (*Server, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return newServer(cfg, newGatewayService()), nil
}

func newServer(cfg Config, service gatewaypb.GatewayServiceServer) *Server {
	grpcServer := grpc.NewServer()
	gatewaypb.RegisterGatewayServiceServer(grpcServer, service)
	return &Server{
		port:       cfg.Port,
		grpcServer: grpcServer,
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
		return fmt.Errorf("listen gateway rpc on port %d: %w", s.port, err)
	}
	s.listener = listener

	go func() {
		if serveErr := s.grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			g.Log().Errorf(context.Background(), "gateway rpc server stopped unexpectedly: %v", serveErr)
		}
	}()
	glog.Infof(context.Background(), "gateway rpc server listening on port %d", s.port)
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
	s.grpcServer.Stop()
}
