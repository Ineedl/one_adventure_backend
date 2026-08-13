package config

import (
	obslog "one_adventure_observability_log"
	metric "one_adventure_observability_metric"
	tracekit "one_adventure_observability_trace/trace"
)

// TraceRuntime 转换为 trace 包使用的运行配置。
func (c Config) TraceRuntime() tracekit.Config {
	return tracekit.Config{Enabled: c.Trace.Enabled, Endpoint: c.Trace.Endpoint, Insecure: c.Trace.Insecure, SampleRatio: c.Trace.SampleRatio}
}

func (c Config) MetricRuntime() metric.Config {
	return metric.Config{Enabled: c.Metric.Enabled, Address: c.Metric.Address}
}

// LogRuntime 转换为 log 包使用的运行配置。
func (c Config) LogRuntime() obslog.Config {
	return obslog.Config{Enabled: c.Log.Enabled, LokiURL: c.Log.LokiURL, BatchSize: c.Log.BatchSize, FlushInterval: c.Log.FlushInterval}
}
