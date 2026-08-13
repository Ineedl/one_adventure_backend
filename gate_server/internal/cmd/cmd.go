package cmd

import (
	"context"

	obsconfig "one_adventure_observability_config"
	obslog "one_adventure_observability_log"
	metric "one_adventure_observability_metric"
	tracekit "one_adventure_observability_trace/trace"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"gate_server/internal/websocket"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start websocket server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			observability, err := obsconfig.Load(ctx, "trace.yaml")
			if err != nil {
				return err
			}
			shutdownTrace, err := tracekit.Init("gate_server", observability.TraceRuntime())
			if err != nil {
				return err
			}
			defer shutdownTrace(context.Background())
			shutdownMetric, err := metric.Init(observability.MetricRuntime())
			if err != nil {
				return err
			}
			defer shutdownMetric(context.Background())
			shutdownLog := obslog.Init("gate_server", obslog.NewInstanceID("gate_server"), observability.LogRuntime())
			defer shutdownLog(context.Background())
			server, err := websocket.New(ctx)
			if err != nil {
				return err
			}
			if err = server.Start(); err != nil {
				return err
			}
			ghttp.Wait()
			return server.Shutdown()
		},
	}
)
