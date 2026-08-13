package tracekit

import (
	"context"
	"io"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

// metadataCarrier 将 gRPC metadata.MD 适配为 OpenTelemetry TextMapCarrier。
type metadataCarrier metadata.MD

// Get 返回指定 metadata key 的第一个值。
func (m metadataCarrier) Get(key string) string {
	values := metadata.MD(m).Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// Set 设置指定 metadata key 的单个值。
func (m metadataCarrier) Set(key, value string) { metadata.MD(m).Set(key, value) }

// Keys 返回 Carrier 中的全部 metadata key。
func (m metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// injectGRPC 将当前 Trace Context 和 Request ID 注入出站 gRPC metadata。
func injectGRPC(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	if md == nil {
		md = metadata.MD{}
	}
	otel.GetTextMapPropagator().Inject(ctx, metadataCarrier(md))
	md.Set("x-trace-id", TraceID(ctx))
	md.Set("x-request-id", RequestID(ctx))
	return metadata.NewOutgoingContext(ctx, md)
}

// extractGRPC 从入站 gRPC metadata 提取 Trace Context 和 Request ID。
// 当 W3C traceparent 缺失时，会回退读取兼容字段 x-trace-id。
func extractGRPC(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	ctx = otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))
	if !trace.SpanContextFromContext(ctx).IsValid() {
		values := md.Get("x-trace-id")
		if len(values) > 0 {
			ctx = ContextWithTraceID(ctx, values[0])
		}
	}
	values := md.Get("x-request-id")
	id := ""
	if len(values) > 0 {
		id = values[0]
	}
	return EnsureRequestID(ctx, id)
}

// UnaryClientInterceptor 为每次一元 gRPC 客户端调用创建 Client Span，并向下游传播上下文。
func UnaryClientInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx, span := Tracer().Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	err := invoker(injectGRPC(ctx), method, req, reply, cc, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, grpcstatus.Code(err).String())
	}
	return err
}

// UnaryServerInterceptor 从一元 gRPC 请求提取父链路，创建 Server Span，并记录调用错误状态。
func UnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx, span := Tracer().Start(extractGRPC(ctx), info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	response, err := handler(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, grpcstatus.Code(err).String())
	}
	return response, err
}

// StreamClientInterceptor 为 gRPC 客户端流创建 Client Span，并在流结束时关闭 Span。
func StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx, span := Tracer().Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
	stream, err := streamer(injectGRPC(ctx), desc, cc, method, opts...)
	if err != nil {
		span.RecordError(err)
		span.End()
		return nil, err
	}
	return &clientStream{ClientStream: stream, span: span}, nil
}

// StreamServerInterceptor 从服务端流的 metadata 提取父链路，并使用包含追踪上下文的流调用业务 Handler。
func StreamServerInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx, span := Tracer().Start(extractGRPC(stream.Context()), info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	return handler(srv, &serverStream{ServerStream: stream, ctx: ctx})
}

// serverStream 包装 grpc.ServerStream，使业务代码通过 Context 获取提取后的追踪上下文。
type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包含当前服务端 Span 和 Request ID 的上下文。
func (s *serverStream) Context() context.Context { return s.ctx }

// clientStream 包装 grpc.ClientStream，负责在接收结束时只关闭一次 Client Span。
type clientStream struct {
	grpc.ClientStream
	span trace.Span
	once sync.Once
}

// CloseSend 关闭客户端发送方向，并将关闭错误记录到 Span。
func (s *clientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		s.span.RecordError(err)
	}
	return err
}

// RecvMsg 接收流消息；遇到 EOF 或错误时结束 Client Span，非 EOF 错误同时写入 Span。
func (s *clientStream) RecvMsg(message any) error {
	err := s.ClientStream.RecvMsg(message)
	if err != nil {
		if err != io.EOF {
			s.span.RecordError(err)
		}
		s.once.Do(func() { s.span.End() })
	}
	return err
}
