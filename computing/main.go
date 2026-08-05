package main

import (
	_ "one_adventure_computing/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"one_adventure_computing/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
