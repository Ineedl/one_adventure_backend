package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// Config contains the gateway gRPC server configuration stored under the
// "rpc.gateway" node.
type Config struct {
	Port         int           `json:"port"`
	PingInterval time.Duration `json:"pingInterval"`
	PingTimeout  time.Duration `json:"pingTimeout"`
}

func loadConfig(ctx context.Context) (Config, error) {
	value, err := g.Cfg().Get(ctx, "rpc.gateway")
	if err != nil {
		return Config{}, fmt.Errorf("read gateway rpc config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("gateway rpc config is required")
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
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.PingInterval <= 0 {
		return fmt.Errorf("ping interval must be greater than zero")
	}
	if c.PingTimeout <= 0 {
		return fmt.Errorf("ping timeout must be greater than zero")
	}
	return nil
}
