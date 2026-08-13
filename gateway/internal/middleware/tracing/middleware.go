package tracing

import (
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"
	tracekit "one_adventure_observability_trace/trace"
)

// Handle extracts W3C trace context or starts a new gateway HTTP trace.
func Handle(request *ghttp.Request) {
	handler := tracekit.HTTPMiddleware("gateway", http.HandlerFunc(func(_ http.ResponseWriter, traced *http.Request) {
		request.SetCtx(traced.Context())
		request.Middleware.Next()
	}))
	handler.ServeHTTP(request.Response.Writer, request.Request)
}
