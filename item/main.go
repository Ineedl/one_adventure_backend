package main

import (
	_ "item/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"item/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
