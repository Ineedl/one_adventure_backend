package main

import (
	_ "gate_server/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"gate_server/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
