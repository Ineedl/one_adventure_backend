// Package tracekit 提供跨 HTTP、gRPC 和 WebSocket 协议的分布式链路追踪能力。
package tracekit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	TraceIDHeader   = "X-Trace-ID"
	RequestIDHeader = "X-Request-ID"
	tracerName      = "one-adventure"
)

// Config 描述 Trace exporter 的运行参数。
type Config struct {
	Enabled     bool
	Endpoint    string
	Insecure    bool
	SampleRatio float64
}

// requestIDKey 是 Request ID 在 context.Context 中使用的私有键类型，避免与其他包的键发生冲突。
type requestIDKey struct{}

// Init 初始化当前进程的 OpenTelemetry TracerProvider 和 W3C Trace Context 传播器。
// 默认向本机 Tempo 的 127.0.0.1:4317 通过 OTLP/gRPC 批量上报；可以使用标准
// OTEL_EXPORTER_OTLP_ENDPOINT 或 OTEL_EXPORTER_OTLP_TRACES_ENDPOINT 环境变量覆盖。
// 返回的关闭函数用于退出前刷新并释放资源。
func Init(serviceName string, cfg Config) (func(context.Context) error, error) {
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, err
	}
	// ParentBased 保留上游采样决定；没有父 Span 时按配置比例采集根链路。
	sampler := trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRatio))
	options := []trace.TracerProviderOption{trace.WithSampler(sampler), trace.WithResource(res)}
	if cfg.Enabled {
		exporterOptions := []otlptracegrpc.Option{otlptracegrpc.WithEndpointURL(cfg.Endpoint)}
		if cfg.Insecure {
			exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
		}
		exporter, exporterErr := otlptracegrpc.New(context.Background(), exporterOptions...)
		if exporterErr != nil {
			return nil, exporterErr
		}
		options = append(options, trace.WithBatcher(exporter))
	}
	provider := trace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return provider.Shutdown, nil
}

// ContextWithTraceID 将兼容格式的 Trace ID 转换为远程父 SpanContext 并写入上下文。
// value 为空或不是合法的 32 位十六进制 Trace ID 时，原上下文保持不变。
func ContextWithTraceID(ctx context.Context, value string) context.Context {
	traceID, err := oteltrace.TraceIDFromHex(value)
	if err != nil {
		return ctx
	}
	var spanBytes [8]byte
	if _, err = rand.Read(spanBytes[:]); err != nil {
		return ctx
	}
	return oteltrace.ContextWithRemoteSpanContext(ctx, oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: traceID, SpanID: oteltrace.SpanID(spanBytes), TraceFlags: oteltrace.FlagsSampled, Remote: true,
	}))
}

// Tracer 返回 tracekit 统一使用的 OpenTelemetry Tracer。
func Tracer() oteltrace.Tracer { return otel.Tracer(tracerName) }

// TraceID 从上下文中的当前 SpanContext 读取 Trace ID。
func TraceID(ctx context.Context) string {
	return oteltrace.SpanContextFromContext(ctx).TraceID().String()
}

// SpanID 从上下文中的当前 SpanContext 读取 Span ID。
func SpanID(ctx context.Context) string {
	return oteltrace.SpanContextFromContext(ctx).SpanID().String()
}

// WithRequestID 将业务请求编号写入上下文。
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID 从上下文读取业务请求编号；不存在时返回空字符串。
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// EnsureRequestID 使用 candidate 作为请求编号；candidate 为空时生成新编号。
func EnsureRequestID(ctx context.Context, candidate string) context.Context {
	if candidate == "" {
		candidate = newID()
	}
	return WithRequestID(ctx, candidate)
}

// ExtractHTTP 从 HTTP Header 提取 W3C traceparent、baggage 和 request_id。
// 当 traceparent 不存在时，会尝试兼容读取 X-Trace-ID。
func ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(header))
	if !oteltrace.SpanContextFromContext(ctx).IsValid() {
		ctx = ContextWithTraceID(ctx, header.Get(TraceIDHeader))
	}
	return EnsureRequestID(ctx, header.Get(RequestIDHeader))
}

// InjectHTTP 将 W3C Trace Context、Trace ID 和 Request ID 注入 HTTP Header。
func InjectHTTP(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
	header.Set(TraceIDHeader, TraceID(ctx))
	header.Set(RequestIDHeader, RequestID(ctx))
}

// newID 使用密码学安全随机数生成 128 位十六进制请求编号。
func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "request-unknown"
	}
	return hex.EncodeToString(value[:])
}
