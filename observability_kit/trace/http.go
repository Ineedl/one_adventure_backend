package tracekit

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware 创建服务端追踪中间件。
// 中间件负责提取上游上下文、创建 Server Span、回写追踪响应头，并将新上下文传给后续 Handler。
func HTTPMiddleware(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ExtractHTTP(r.Context(), r.Header)
		ctx, span := Tracer().Start(ctx, r.Method+" "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(
			attribute.String("service.name", serviceName), attribute.String("http.request.method", r.Method), attribute.String("url.path", r.URL.Path),
		))
		defer span.End()
		InjectHTTP(ctx, w.Header())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// transport 包装底层 RoundTripper，为每次 HTTP 出站请求创建 Client Span 并注入追踪头。
type transport struct{ base http.RoundTripper }

// RoundTrip 实现 http.RoundTripper，是 HTTP Client 实际发送请求时的追踪入口。
func (t transport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx, span := Tracer().Start(request.Context(), request.Method+" "+request.URL.Host, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	clone := request.Clone(ctx)
	InjectHTTP(ctx, clone.Header)
	response, err := t.base.RoundTrip(clone)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return response, err
}

// NewHTTPClient 返回能够自动创建 Client Span 并传播 Trace Context 的 HTTP Client。
// 该函数会浅拷贝传入 Client，不会修改调用方持有的原对象；base 为 nil 时使用默认配置。
func NewHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	clone := *base
	baseTransport := clone.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	clone.Transport = transport{base: baseTransport}
	return &clone
}
