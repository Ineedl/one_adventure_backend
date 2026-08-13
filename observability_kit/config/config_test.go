package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFileThenEnvironmentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.yaml")
	data := []byte("trace:\n  enabled: true\n  endpoint: http://file:4317\n  insecure: false\n  sampleRatio: 0.5\nlog:\n  enabled: true\n  lokiUrl: http://file:3100\n  batchSize: 20\n  flushInterval: 2s\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://env:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "1")
	t.Setenv("LOKI_URL", "http://env:3100")
	t.Setenv("LOKI_BATCH_SIZE", "30")
	t.Setenv("LOKI_FLUSH_INTERVAL", "3s")
	cfg, err := Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Trace.Endpoint != "http://env:4317" || !cfg.Trace.Insecure || cfg.Trace.SampleRatio != 1 {
		t.Fatalf("trace = %#v", cfg.Trace)
	}
	if cfg.Log.LokiURL != "http://env:3100" || cfg.Log.BatchSize != 30 || cfg.Log.FlushInterval != 3*time.Second {
		t.Fatalf("log = %#v", cfg.Log)
	}
}

func TestEmptyEnvironmentDoesNotOverrideFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.yaml")
	data := []byte("trace:\n  enabled: true\n  endpoint: http://file:4317\n  insecure: true\n  sampleRatio: 1\nlog:\n  enabled: true\n  lokiUrl: http://file:3100\n  batchSize: 20\n  flushInterval: 2s\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("LOKI_URL", "")
	cfg, err := Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Trace.Endpoint != "http://file:4317" || cfg.Log.LokiURL != "http://file:3100" {
		t.Fatalf("config = %#v", cfg)
	}
}
