package cmd

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"one_adventure_gateway/internal/router"
	gatewayrpc "one_adventure_gateway/internal/rpc/gateway"
	"one_adventure_gateway/internal/websocket"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			wsServer, err := websocket.New(ctx)
			if err != nil {
				return err
			}
			rpcServer, err := gatewayrpc.New(ctx)
			if err != nil {
				return err
			}
			s, err := router.New(ctx, rpcServer)
			if err != nil {
				return err
			}
			if err = s.Start(); err != nil {
				return err
			}
			if err = wsServer.Start(); err != nil {
				_ = s.Shutdown()
				return err
			}
			if err = rpcServer.Start(); err != nil {
				_ = wsServer.Shutdown()
				_ = s.Shutdown()
				return err
			}

			// 当前进程包含 HTTP、WebSocket 和 Gateway gRPC 服务。
			ghttp.Wait()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return rpcServer.Shutdown(shutdownCtx)
		},
	}
)
