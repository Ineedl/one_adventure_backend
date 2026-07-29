package websocket

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

// WsConfig contains the WebSocket server configuration stored under the
// "websocket" node in manifest/config/config.yaml.
type WsConfig struct {
	Port            int    `json:"port"`
	Path            string `json:"path"`
	ReadBufferSize  int    `json:"readBufferSize"`
	WriteBufferSize int    `json:"writeBufferSize"`
	MaxMessageSize  int64  `json:"maxMessageSize"`
}

func loadConfig(ctx context.Context) (WsConfig, error) {
	value, err := g.Cfg().Get(ctx, "websocket")
	if err != nil {
		return WsConfig{}, fmt.Errorf("read websocket config: %w", err)
	}
	if value.IsEmpty() {
		return WsConfig{}, fmt.Errorf("websocket config is required")
	}

	var cfg WsConfig
	if err = value.Scan(&cfg); err != nil {
		return WsConfig{}, fmt.Errorf("parse websocket config: %w", err)
	}
	if err = cfg.validate(); err != nil {
		return WsConfig{}, fmt.Errorf("validate websocket config: %w", err)
	}
	return cfg, nil
}

func (c WsConfig) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.Path == "" || !strings.HasPrefix(c.Path, "/") {
		return fmt.Errorf("path must start with /")
	}
	if c.ReadBufferSize <= 0 {
		return fmt.Errorf("readBufferSize must be greater than 0")
	}
	if c.WriteBufferSize <= 0 {
		return fmt.Errorf("writeBufferSize must be greater than 0")
	}
	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("maxMessageSize must be greater than 0")
	}
	return nil
}
