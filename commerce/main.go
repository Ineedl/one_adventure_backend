package main

import (
	_ "commerce/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"commerce/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
