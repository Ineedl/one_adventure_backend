package main

import (
	_ "promotion/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"promotion/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
