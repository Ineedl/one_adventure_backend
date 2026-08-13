// Package metric exposes process resource metrics in Prometheus format.
package metric

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config controls the Prometheus scrape endpoint.
type Config struct {
	Enabled bool
	Address string
}

// Init starts a dedicated HTTP server exposing /metrics. The returned function
// gracefully stops it. The standard process collector supplies CPU and memory
// statistics (and process network byte counters where supported).
func Init(cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.Address == "" {
		return nil, errors.New("metric address is required when metrics are enabled")
	}
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("listen for Prometheus metrics: %w", err)
	}
	go func() { _ = server.Serve(listener) }()
	return server.Shutdown, nil
}
