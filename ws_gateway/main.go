package main

import (
	_ "ws_gateway/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"ws_gateway/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
