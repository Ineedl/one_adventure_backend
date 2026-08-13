// Package accesslog provides HTTP access logging for the gateway.
package accesslog

import (
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	obslog "one_adventure_observability_log"
)

// Handle logs both the arrival and completion of every gateway HTTP request.
func Handle(request *ghttp.Request) {
	startedAt := time.Now()
	ctx := request.Context()
	method := request.Method
	uri := request.URL.RequestURI()
	clientIP := request.GetClientIp()

	obslog.Info(ctx, "http request started", map[string]any{"method": method, "uri": uri, "client_ip": clientIP})

	request.Middleware.Next()

	obslog.Info(ctx, "http request completed", map[string]any{"method": method, "uri": uri, "client_ip": clientIP, "status": request.Response.Status, "duration_ms": time.Since(startedAt).Milliseconds()})
}
