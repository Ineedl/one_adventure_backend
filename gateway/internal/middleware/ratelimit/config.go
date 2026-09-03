package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/os/gcfg"
)

type Config struct {
	TokenBucket TokenBucketConfig `json:"tokenBucket"`
	LeakyBucket LeakyBucketConfig `json:"leakyBucket"`
	UserWindow  UserWindowConfig  `json:"userWindow"`
}

type TokenBucketConfig struct {
	Enabled  bool    `json:"enabled"`
	Capacity int     `json:"capacity"`
	Rate     float64 `json:"rate"` // tokens per second
}

type LeakyBucketConfig struct {
	Enabled  bool    `json:"enabled"`
	Capacity int     `json:"capacity"`
	Rate     float64 `json:"rate"` // requests leaked per second
}

type UserWindowConfig struct {
	Enabled bool          `json:"enabled"`
	Limit   int           `json:"limit"`
	Window  time.Duration `json:"window"`
}

func loadConfig(ctx context.Context) (Config, error) {
	adapter, err := gcfg.NewAdapterFile("rate_limit.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("create rate limit config adapter: %w", err)
	}
	value, err := gcfg.NewWithAdapter(adapter).Get(ctx, ".")
	if err != nil {
		return Config{}, fmt.Errorf("read rate limit config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("rate limit config is required")
	}
	var config Config
	if err = value.Scan(&config); err != nil {
		return Config{}, fmt.Errorf("parse rate limit config: %w", err)
	}
	if config.TokenBucket.Enabled && (config.TokenBucket.Capacity <= 0 || config.TokenBucket.Rate <= 0) {
		return Config{}, fmt.Errorf("token bucket capacity and rate must be positive")
	}
	if config.LeakyBucket.Enabled && (config.LeakyBucket.Capacity <= 0 || config.LeakyBucket.Rate <= 0) {
		return Config{}, fmt.Errorf("leaky bucket capacity and rate must be positive")
	}
	if config.UserWindow.Enabled && (config.UserWindow.Limit <= 0 || config.UserWindow.Window <= 0) {
		return Config{}, fmt.Errorf("user window limit and window must be positive")
	}
	return config, nil
}
