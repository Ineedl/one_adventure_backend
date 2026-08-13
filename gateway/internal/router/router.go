package router

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"one_adventure_gateway/internal/middleware/accesslog"
	"one_adventure_gateway/internal/middleware/auth"
	"one_adventure_gateway/internal/middleware/tracing"
	"one_adventure_gateway/internal/service"
)

// New configures and returns the gateway HTTP server.
func New(ctx context.Context, resolver service.ServiceResolver) (*ghttp.Server, error) {
	authMiddleware, err := auth.New(ctx)
	if err != nil {
		return nil, err
	}
	server := g.Server()
	handler := NewHandler(resolver, DefaultRouteTable())
	server.Group("/", func(group *ghttp.RouterGroup) {
		group.Middleware(tracing.Handle, accesslog.Handle, authMiddleware.Handle)
		group.ALL("/*path", handler.Handle)
	})
	return server, nil
}
