package main

import (
	_ "pay/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"pay/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
