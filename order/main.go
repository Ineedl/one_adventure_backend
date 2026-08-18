package main

import (
	_ "order/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"order/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
