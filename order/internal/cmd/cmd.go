package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"google.golang.org/grpc"
	obsconfig "one_adventure_observability_config"
	obslog "one_adventure_observability_log"
	metric "one_adventure_observability_metric"
	tracekit "one_adventure_observability_trace/trace"
	orderpb "one_adventure_rpc/proto/order"
	kafkakit "one_adventure_servicekit/kafka"
	"order/internal/compensation"
	"order/internal/consumer/promotionorder"
	"order/internal/controller/hello"
	orderrpc "order/internal/rpc/order"
	ordertimeout "order/internal/timeout"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			observability, err := obsconfig.Load(ctx, "trace.yaml")
			if err != nil {
				return err
			}
			shutdownTrace, err := tracekit.Init("order", observability.TraceRuntime())
			if err != nil {
				return err
			}
			defer shutdownTrace(context.Background())
			shutdownMetric, err := metric.Init(observability.MetricRuntime())
			if err != nil {
				return err
			}
			defer shutdownMetric(context.Background())
			shutdownLog := obslog.Init("order", obslog.NewInstanceID("order"), observability.LogRuntime())
			defer shutdownLog(context.Background())
			consumer, err := promotionorder.New(ctx)
			if err != nil {
				return err
			}
			consumerCtx, cancelConsumer := context.WithCancel(context.Background())
			defer cancelConsumer()
			defer consumer.Close()
			kafkaConfig, err := kafkakit.LoadConfig(ctx)
			if err != nil {
				return err
			}
			compensationProducer := kafkakit.NewProducer(kafkaConfig)
			defer compensationProducer.Close()
			compensationPublisher := compensation.New(compensationProducer)
			grpcService := orderrpc.NewService(compensationPublisher)
			go func() {
				if runErr := consumer.Run(consumerCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
					g.Log().Errorf(context.Background(), "promotion order consumer stopped: %v", runErr)
				}
			}()
			grpcServer := grpc.NewServer()
			orderpb.RegisterOrderServiceServer(grpcServer, grpcService)
			grpcListener, err := net.Listen("tcp", ":9009")
			if err != nil {
				return fmt.Errorf("listen order grpc: %w", err)
			}
			go func() {
				if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
					obslog.Error(context.Background(), "order grpc stopped", map[string]any{"error": serveErr.Error()})
				}
			}()
			defer grpcServer.GracefulStop()
			timeoutTask := ordertimeout.New(compensationPublisher)
			go func() {
				if runErr := timeoutTask.Run(consumerCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
					obslog.Error(context.Background(), "order timeout task stopped", map[string]any{"error": runErr.Error()})
				}
			}()
			s := g.Server()
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					hello.NewV1(),
				)
			})
			s.Run()
			return nil
		},
	}
)
