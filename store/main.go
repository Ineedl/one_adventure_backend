package main

import (
	_ "store/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"store/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
