package main

import (
	_ "server_manager/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"server_manager/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
