package promotion

import (
	"context"
	"errors"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	obslog "one_adventure_observability_log"
	tracekit "one_adventure_observability_trace/trace"
	pb "one_adventure_rpc/proto/promotion"
	"one_adventure_servicekit/discovery"
	kafkakit "one_adventure_servicekit/kafka"
)

type Server struct {
	port      int
	grpc      *grpc.Server
	registrar *discovery.Registrar
	producer  *kafkakit.Producer
	stop      context.CancelFunc
}

func New(ctx context.Context) (*Server, error) {
	c, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	kc, err := kafkakit.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	r, err := discovery.NewRegistrar(c.discoveryConfig(), c.registration())
	if err != nil {
		return nil, err
	}
	p := kafkakit.NewProducer(kc)
	gs := grpc.NewServer(
		grpc.ChainUnaryInterceptor(tracekit.UnaryServerInterceptor, errorLoggingUnaryInterceptor),
		grpc.StreamInterceptor(tracekit.StreamServerInterceptor),
	)
	pb.RegisterPromotionServiceServer(gs, newService(p))
	return &Server{port: c.Port, grpc: gs, registrar: r, producer: p}, nil
}

func errorLoggingUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	response, err := handler(ctx, req)
	if err != nil {
		rpcStatus := status.Convert(err)
		obslog.Error(ctx, "promotion rpc request failed", map[string]any{
			"method": info.FullMethod,
			"code":   rpcStatus.Code().String(),
			"error":  rpcStatus.Message(),
		})
	}
	return response, err
}
func (s *Server) InstanceID() string { return s.registrar.InstanceID() }
func (s *Server) Start() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	go func() {
		if err := s.registrar.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			obslog.Error(ctx, "promotion service registrar stopped unexpectedly", map[string]any{"error": err.Error()})
		}
	}()
	go func() {
		if serveErr := s.grpc.Serve(l); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			obslog.Error(context.Background(), "promotion grpc server stopped unexpectedly", map[string]any{"error": serveErr.Error()})
		}
	}()
	return nil
}
func (s *Server) Shutdown(ctx context.Context) error {
	if s.stop != nil {
		s.stop()
	}
	done := make(chan struct{})
	go func() { s.grpc.GracefulStop(); close(done) }()
	select {
	case <-done:
		return s.producer.Close()
	case <-ctx.Done():
		s.grpc.Stop()
		return ctx.Err()
	}
}
