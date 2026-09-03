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
	paypb "one_adventure_rpc/proto/pay"
	kafkakit "one_adventure_servicekit/kafka"

	"pay/internal/consumer/refundreport"
	"pay/internal/controller/hello"
	payrefund "pay/internal/refund"
	"pay/internal/report"
	payrpc "pay/internal/rpc/pay"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			kafkaConfig, err := kafkakit.LoadConfig(ctx)
			if err != nil {
				return err
			}
			producer := kafkakit.NewProducer(kafkaConfig)
			defer producer.Close()
			refundConsumer, err := refundreport.New(ctx)
			if err != nil {
				return err
			}
			defer refundConsumer.Close()
			workerCtx, cancelWorkers := context.WithCancel(context.Background())
			defer cancelWorkers()
			go func() {
				if runErr := refundConsumer.Run(workerCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
					g.Log().Errorf(context.Background(), "refund report consumer stopped: %v", runErr)
				}
			}()
			refundTask := payrefund.New(payrefund.MockGateway{})
			go func() {
				if runErr := refundTask.Run(workerCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
					g.Log().Errorf(context.Background(), "refund task stopped: %v", runErr)
				}
			}()

			grpcServer := grpc.NewServer()
			paypb.RegisterPayServiceServer(grpcServer, payrpc.NewService(report.New(producer)))
			grpcListener, err := net.Listen("tcp", ":9010")
			if err != nil {
				return fmt.Errorf("listen pay grpc: %w", err)
			}
			go func() {
				if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
					g.Log().Errorf(context.Background(), "pay grpc stopped: %v", serveErr)
				}
			}()
			defer grpcServer.GracefulStop()
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
