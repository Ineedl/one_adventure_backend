package tracekit

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TestHTTPTraceContextRoundTrip 验证 HTTP 注入和提取后 Trace ID、Request ID 保持一致。
func TestHTTPTraceContextRoundTrip(t *testing.T) {
	shutdown, err := Init("test", Config{SampleRatio: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown(context.Background())
	ctx, span := Tracer().Start(EnsureRequestID(context.Background(), "request-1"), "parent")
	defer span.End()
	header := http.Header{}
	InjectHTTP(ctx, header)
	extracted := ExtractHTTP(context.Background(), header)
	if TraceID(extracted) != TraceID(ctx) {
		t.Fatalf("trace id = %s, want %s", TraceID(extracted), TraceID(ctx))
	}
	if RequestID(extracted) != "request-1" {
		t.Fatalf("request id = %q", RequestID(extracted))
	}
}

// TestLegacyTraceIDFallback 验证没有 traceparent 时仍可继承旧系统传入的 X-Trace-ID。
func TestLegacyTraceIDFallback(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	header := http.Header{}
	header.Set(TraceIDHeader, "4bf92f3577b34da6a3ce929d0e0e4736")
	ctx := ExtractHTTP(context.Background(), header)
	if TraceID(ctx) != header.Get(TraceIDHeader) {
		t.Fatalf("trace id = %s", TraceID(ctx))
	}
}
