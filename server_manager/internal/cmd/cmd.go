package cmd

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	obsconfig "one_adventure_observability_config"
	obslog "one_adventure_observability_log"
	metric "one_adventure_observability_metric"
	tracekit "one_adventure_observability_trace/trace"

	"server_manager/internal/controller/hello"
	servermanagerrpc "server_manager/internal/rpc/server_manager"
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
			shutdownTrace, err := tracekit.Init("server_manager", observability.TraceRuntime())
			if err != nil {
				return err
			}
			defer shutdownTrace(context.Background())
			shutdownMetric, err := metric.Init(observability.MetricRuntime())
			if err != nil {
				return err
			}
			defer shutdownMetric(context.Background())
			rpcServer, err := servermanagerrpc.New(ctx)
			if err != nil {
				return err
			}
			shutdownLog := obslog.Init("server_manager", rpcServer.InstanceID(), observability.LogRuntime())
			defer shutdownLog(context.Background())
			s := g.Server()
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					hello.NewV1(),
				)
			})
			if err = s.Start(); err != nil {
				return err
			}
			if err = rpcServer.Start(); err != nil {
				_ = s.Shutdown()
				return err
			}
			ghttp.Wait()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return rpcServer.Shutdown(shutdownCtx)
		},
	}
)
