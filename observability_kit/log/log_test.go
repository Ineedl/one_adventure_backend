package log

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	tracekit "one_adventure_observability_trace/trace"
)

// TestLoggerPushesTraceLinkedJSON 验证日志正文携带 Trace 关联字段且 Loki 标签不包含高基数 Trace ID。
func TestLoggerPushesTraceLinkedJSON(t *testing.T) {
	requests := make(chan pushRequest, 1)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		var payload pushRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("decode Loki payload: %v", err)
		}
		requests <- payload
		return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}

	shutdownTrace, err := tracekit.Init("log-test", tracekit.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer shutdownTrace(context.Background())
	ctx, span := tracekit.Tracer().Start(tracekit.EnsureRequestID(context.Background(), "request-1"), "test")
	defer span.End()

	logger := New(Config{Enabled: true, ServiceName: "user", InstanceID: "user-1", LokiURL: "http://loki:3100", Client: client, BatchSize: 1, FlushInterval: time.Hour})
	logger.Info(ctx, "login", map[string]any{"username": "alice"})
	select {
	case payload := <-requests:
		if len(payload.Streams) != 1 || payload.Streams[0].Stream["service"] != "user" {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.Streams[0].Stream["instance_id"] != "user-1" {
			t.Fatalf("instance label = %q", payload.Streams[0].Stream["instance_id"])
		}
		if _, exists := payload.Streams[0].Stream["trace_id"]; exists {
			t.Fatal("trace_id must not be a Loki label")
		}
		var line map[string]any
		if err = json.Unmarshal([]byte(payload.Streams[0].Values[0][1]), &line); err != nil {
			t.Fatal(err)
		}
		if line["trace_id"] != tracekit.TraceID(ctx) || line["request_id"] != "request-1" || line["instance_id"] != "user-1" {
			t.Fatalf("line = %#v", line)
		}
	case <-time.After(time.Second):
		t.Fatal("Loki request was not received")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
