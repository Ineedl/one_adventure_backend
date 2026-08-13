package gateway

import (
	"context"
	"fmt"
	"time"

	"one_adventure_servicekit/discovery"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
)

// Config contains the gateway RPC configuration loaded from rpc.yaml.
type Config struct {
	Port int        `json:"port"`
	Etcd EtcdConfig `json:"etcd"`
}

type EtcdConfig struct {
	Endpoints     []string      `json:"endpoints"`
	DialTimeout   time.Duration `json:"dialTimeout"`
	LeaseTTL      int64         `json:"leaseTtl"`
	ServerName    string        `json:"serverName"`
	InstanceID    string        `json:"instanceId"`
	Address       string        `json:"address"`
	HTTPPort      int           `json:"httpPort"`
	WatchServices []string      `json:"watchServices"`
}

func loadConfig(ctx context.Context) (Config, error) {
	adapter, err := gcfg.NewAdapterFile("rpc.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("create gateway rpc config adapter: %w", err)
	}
	value, err := gcfg.NewWithAdapter(adapter).Get(ctx, ".")
	if err != nil {
		return Config{}, fmt.Errorf("read gateway rpc config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("rpc config is required")
	}

	var cfg Config
	if err = value.Scan(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse gateway rpc config: %w", err)
	}
	if err = cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate gateway rpc config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	return c.discoveryConfig().Validate()
}

func (c Config) discoveryConfig() discovery.Config {
	return discovery.Config{Endpoints: c.Etcd.Endpoints, DialTimeout: c.Etcd.DialTimeout, DebugLog: debugLog, ErrorLog: errorLog}
}

func debugLog(message string, args ...any) {
	g.Log().Debug(context.Background(), append([]any{message}, args...)...)
}

func errorLog(message string, args ...any) {
	g.Log().Error(context.Background(), append([]any{message}, args...)...)
}
