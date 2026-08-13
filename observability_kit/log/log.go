// Package log 提供带 Trace 关联和 Loki 上报能力的结构化日志。
package log

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	tracekit "one_adventure_observability_trace/trace"
)

type contextKey struct{}

// NewInstanceID 根据服务名、主机名和进程 ID 生成当前进程的实例唯一标识。
func NewInstanceID(serviceName string) string {
	hostname, _ := os.Hostname()
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", serviceName, hostname, os.Getpid())))
	return serviceName + "_" + hex.EncodeToString(sum[:6])
}

type Config struct {
	Enabled       bool
	ServiceName   string
	InstanceID    string
	LokiURL       string
	Labels        map[string]string
	Client        *http.Client
	BatchSize     int
	FlushInterval time.Duration
}

type Logger struct {
	cfg     Config
	client  *http.Client
	mu      sync.Mutex
	entries []entry
	stop    chan struct{}
	done    chan struct{}
}

var defaultLogger = New(Config{ServiceName: "unknown", InstanceID: "unknown"})

type entry struct {
	timestamp string
	line      string
}

type pushRequest struct {
	Streams []stream `json:"streams"`
}
type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

// New 创建结构化日志器；LokiURL 为空时仍输出 stdout，但不会远程上报。
func New(cfg Config) *Logger {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "unknown"
	}
	if cfg.InstanceID == "" {
		cfg.InstanceID = "unknown"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	l := &Logger{cfg: cfg, client: client, entries: make([]entry, 0, cfg.BatchSize), stop: make(chan struct{}), done: make(chan struct{})}
	go l.flushLoop()
	return l
}

// Init 使用服务名和实例唯一标识初始化进程级默认日志器。
// LOKI_URL 未配置时默认上报本机 http://127.0.0.1:3100。
func Init(serviceName, instanceID string, cfg Config) func(context.Context) error {
	cfg.ServiceName, cfg.InstanceID = serviceName, instanceID
	if !cfg.Enabled {
		cfg.LokiURL = ""
	}
	defaultLogger = New(cfg)
	return defaultLogger.Shutdown
}

// Info 使用进程级默认日志器记录 INFO 日志。
func Info(ctx context.Context, message string, fields ...map[string]any) {
	defaultLogger.Info(ctx, message, fields...)
}

// Warn 使用进程级默认日志器记录 WARN 日志。
func Warn(ctx context.Context, message string, fields ...map[string]any) {
	defaultLogger.Warn(ctx, message, fields...)
}

// Error 使用进程级默认日志器记录 ERROR 日志。
func Error(ctx context.Context, message string, fields ...map[string]any) {
	defaultLogger.Error(ctx, message, fields...)
}

// Shutdown 刷新剩余日志并停止后台批量发送。
func (l *Logger) Shutdown(ctx context.Context) error {
	close(l.stop)
	select {
	case <-l.done:
		return l.Flush(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// With 返回携带结构化字段的日志上下文。
func With(ctx context.Context, fields map[string]any) context.Context {
	return context.WithValue(ctx, contextKey{}, fields)
}

// Info 记录 INFO 级别日志。
func (l *Logger) Info(ctx context.Context, message string, fields ...map[string]any) {
	l.write(ctx, "INFO", message, fields...)
}

// Warn 记录 WARN 级别日志。
func (l *Logger) Warn(ctx context.Context, message string, fields ...map[string]any) {
	l.write(ctx, "WARN", message, fields...)
}

// Error 记录 ERROR 级别日志。
func (l *Logger) Error(ctx context.Context, message string, fields ...map[string]any) {
	l.write(ctx, "ERROR", message, fields...)
}

func (l *Logger) write(ctx context.Context, level, message string, extra ...map[string]any) {
	fields := map[string]any{"timestamp": time.Now().Format(time.RFC3339Nano), "level": level, "service": l.cfg.ServiceName, "instance_id": l.cfg.InstanceID, "trace_id": tracekit.TraceID(ctx), "span_id": tracekit.SpanID(ctx), "request_id": tracekit.RequestID(ctx), "message": message}
	if value, ok := ctx.Value(contextKey{}).(map[string]any); ok {
		for k, v := range value {
			fields[k] = v
		}
	}
	for _, set := range extra {
		for k, v := range set {
			fields[k] = v
		}
	}
	data, _ := json.Marshal(fields)
	line := string(data)
	fmt.Fprintln(os.Stdout, line)
	l.mu.Lock()
	l.entries = append(l.entries, entry{timestamp: fmt.Sprintf("%d", time.Now().UnixNano()), line: line})
	flush := len(l.entries) >= l.cfg.BatchSize
	l.mu.Unlock()
	if flush {
		_ = l.Flush(context.Background())
	}
}

// Flush 将当前批次日志发送到 Loki Push API。
func (l *Logger) Flush(ctx context.Context) error {
	if l.cfg.LokiURL == "" {
		l.mu.Lock()
		l.entries = nil
		l.mu.Unlock()
		return nil
	}
	l.mu.Lock()
	entries := l.entries
	l.entries = nil
	l.mu.Unlock()
	if len(entries) == 0 {
		return nil
	}
	values := make([][2]string, len(entries))
	for i, e := range entries {
		values[i] = [2]string{e.timestamp, e.line}
	}
	labels := map[string]string{"service": l.cfg.ServiceName, "instance_id": l.cfg.InstanceID}
	for k, v := range l.cfg.Labels {
		labels[k] = v
	}
	body, _ := json.Marshal(pushRequest{Streams: []stream{{Stream: labels, Values: values}}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, l.cfg.LokiURL+"/loki/api/v1/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("loki push returned %s", resp.Status)
	}
	return nil
}
func (l *Logger) flushLoop() {
	ticker := time.NewTicker(l.cfg.FlushInterval)
	defer ticker.Stop()
	defer close(l.done)
	for {
		select {
		case <-ticker.C:
			_ = l.Flush(context.Background())
		case <-l.stop:
			return
		}
	}
}
