package cmd

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"one_adventure_computing/internal/controller/hello"
	computingrpc "one_adventure_computing/internal/rpc/computing"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start computing servers",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					hello.NewV1(),
				)
			})

			rpcServer, err := computingrpc.New(ctx)
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

			ghttp.Wait()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return rpcServer.Shutdown(shutdownCtx)
		},
	}
)
