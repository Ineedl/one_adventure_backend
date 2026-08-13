// Package config 负责加载 Trace 和日志上报的统一配置。
package config

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/gogf/gf/v2/os/gcfg"
)

// Config 是服务可观测性配置文件的根结构。
type Config struct {
	Trace  TraceConfig  `json:"trace"`
	Log    LogConfig    `json:"log"`
	Metric MetricConfig `json:"metric"`
}

type MetricConfig struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
}

// TraceConfig 描述 OpenTelemetry Trace 上报参数。
type TraceConfig struct {
	Enabled     bool    `json:"enabled"`
	Endpoint    string  `json:"endpoint"`
	Insecure    bool    `json:"insecure"`
	SampleRatio float64 `json:"sampleRatio"`
}

// LogConfig 描述 Loki 日志上报参数。
type LogConfig struct {
	Enabled       bool          `json:"enabled"`
	LokiURL       string        `json:"lokiUrl"`
	BatchSize     int           `json:"batchSize"`
	FlushInterval time.Duration `json:"flushInterval"`
}

// Load 先读取配置文件，再用非空环境变量覆盖对应配置。
func Load(ctx context.Context, path string) (Config, error) {
	adapter, err := gcfg.NewAdapterFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("create observability config adapter: %w", err)
	}
	value, err := gcfg.NewWithAdapter(adapter).Get(ctx, ".")
	if err != nil {
		return Config{}, fmt.Errorf("read observability config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("observability config %q is required", path)
	}
	var cfg Config
	if err = value.Scan(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse observability config: %w", err)
	}
	if err = applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.Trace.Enabled && cfg.Trace.Endpoint == "" {
		return Config{}, fmt.Errorf("trace endpoint is required when trace is enabled")
	}
	if math.IsNaN(cfg.Trace.SampleRatio) || cfg.Trace.SampleRatio < 0 || cfg.Trace.SampleRatio > 1 {
		return Config{}, fmt.Errorf("trace sample ratio must be between 0 and 1")
	}
	if cfg.Log.Enabled && cfg.Log.LokiURL == "" {
		return Config{}, fmt.Errorf("Loki URL is required when log reporting is enabled")
	}
	if cfg.Metric.Enabled && cfg.Metric.Address == "" {
		return Config{}, fmt.Errorf("metric address is required when metrics are enabled")
	}
	return cfg, nil
}

func applyEnvironment(cfg *Config) error {
	if value := firstNonEmpty(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"), os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); value != "" {
		cfg.Trace.Endpoint = value
	}
	if value := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse OTEL_EXPORTER_OTLP_INSECURE: %w", err)
		}
		cfg.Trace.Insecure = parsed
	}
	if value := os.Getenv("TRACE_ENABLED"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse TRACE_ENABLED: %w", err)
		}
		cfg.Trace.Enabled = parsed
	}
	if value := firstNonEmpty(os.Getenv("OTEL_TRACES_SAMPLER_ARG"), os.Getenv("TRACE_SAMPLE_RATIO")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse trace sample ratio: %w", err)
		}
		cfg.Trace.SampleRatio = parsed
	}
	if value := os.Getenv("LOKI_URL"); value != "" {
		cfg.Log.LokiURL = value
	}
	if value := os.Getenv("LOG_REPORT_ENABLED"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse LOG_REPORT_ENABLED: %w", err)
		}
		cfg.Log.Enabled = parsed
	}
	if value := os.Getenv("LOKI_BATCH_SIZE"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse LOKI_BATCH_SIZE: %w", err)
		}
		cfg.Log.BatchSize = parsed
	}
	if value := os.Getenv("LOKI_FLUSH_INTERVAL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse LOKI_FLUSH_INTERVAL: %w", err)
		}
		cfg.Log.FlushInterval = parsed
	}
	if value := os.Getenv("METRIC_ENABLED"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse METRIC_ENABLED: %w", err)
		}
		cfg.Metric.Enabled = parsed
	}
	if value := os.Getenv("METRIC_ADDRESS"); value != "" {
		cfg.Metric.Address = value
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
