package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/os/gcfg"
)

type Config struct {
	PublicKeyPath string   `json:"publicKeyPath"`
	Issuer        string   `json:"issuer"`
	Whitelist     []string `json:"whitelist"`
}

func loadConfig(ctx context.Context) (Config, error) {
	adapter, err := gcfg.NewAdapterFile("auth.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("create auth config adapter: %w", err)
	}
	value, err := gcfg.NewWithAdapter(adapter).Get(ctx, ".")
	if err != nil {
		return Config{}, fmt.Errorf("read auth config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("auth config is required")
	}
	var config Config
	if err = value.Scan(&config); err != nil {
		return Config{}, fmt.Errorf("parse auth config: %w", err)
	}
	config.PublicKeyPath = strings.TrimSpace(config.PublicKeyPath)
	config.Issuer = strings.TrimSpace(config.Issuer)
	if config.PublicKeyPath == "" {
		return Config{}, fmt.Errorf("auth public key path is required")
	}
	for _, pattern := range config.Whitelist {
		if err = validatePattern(pattern); err != nil {
			return Config{}, fmt.Errorf("invalid auth whitelist pattern %q: %w", pattern, err)
		}
	}
	return config, nil
}
