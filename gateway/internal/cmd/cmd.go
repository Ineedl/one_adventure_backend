package cmd

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	obsconfig "one_adventure_observability_config"
	obslog "one_adventure_observability_log"
	metric "one_adventure_observability_metric"
	tracekit "one_adventure_observability_trace/trace"

	"one_adventure_gateway/internal/router"
	gatewayrpc "one_adventure_gateway/internal/rpc/gateway"
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
			shutdownTrace, err := tracekit.Init("gateway", observability.TraceRuntime())
			if err != nil {
				return err
			}
			defer shutdownTrace(context.Background())
			shutdownMetric, err := metric.Init(observability.MetricRuntime())
			if err != nil {
				return err
			}
			defer shutdownMetric(context.Background())
			rpcServer, err := gatewayrpc.New(ctx)
			if err != nil {
				return err
			}
			shutdownLog := obslog.Init("gateway", rpcServer.InstanceID(), observability.LogRuntime())
			defer shutdownLog(context.Background())
			s, err := router.New(ctx, rpcServer)
			if err != nil {
				return err
			}
			if err = s.Start(); err != nil {
				return err
			}
			if err = rpcServer.Start(); err != nil {
				_ = s.Shutdown()
				return err
			}

			// 当前进程包含 HTTP 和 Gateway gRPC 服务。
			ghttp.Wait()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return rpcServer.Shutdown(shutdownCtx)
		},
	}
)
