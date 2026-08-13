package server_manager

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"one_adventure_servicekit/discovery"
)

type Config struct {
	Port int        `json:"port"`
	Etcd EtcdConfig `json:"etcd"`
}

type EtcdConfig struct {
	Endpoints   []string      `json:"endpoints"`
	DialTimeout time.Duration `json:"dialTimeout"`
	LeaseTTL    int64         `json:"leaseTtl"`
	ServerName  string        `json:"serverName"`
	InstanceID  string        `json:"instanceId"`
	Address     string        `json:"address"`
	HTTPPort    int           `json:"httpPort"`
}

func loadConfig(ctx context.Context) (Config, error) {
	adapter, err := gcfg.NewAdapterFile("rpc.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("create server manager rpc config adapter: %w", err)
	}
	value, err := gcfg.NewWithAdapter(adapter).Get(ctx, ".")
	if err != nil {
		return Config{}, fmt.Errorf("read server manager rpc config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("rpc config is required")
	}
	var config Config
	if err = value.Scan(&config); err != nil {
		return Config{}, fmt.Errorf("parse server manager rpc config: %w", err)
	}
	if err = config.validate(); err != nil {
		return Config{}, fmt.Errorf("validate server manager rpc config: %w", err)
	}
	return config, nil
}

func (c Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if err := c.discoveryConfig().Validate(); err != nil {
		return err
	}
	return c.registration().Validate()
}

func (c Config) discoveryConfig() discovery.Config {
	return discovery.Config{
		Endpoints: c.Etcd.Endpoints, DialTimeout: c.Etcd.DialTimeout,
		DebugLog: debugLog, ErrorLog: errorLog,
	}
}

func (c Config) registration() discovery.Registration {
	return discovery.Registration{
		ServerName: c.Etcd.ServerName,
		InstanceID: c.Etcd.InstanceID,
		LeaseTTL:   c.Etcd.LeaseTTL,
		Instance: discovery.Instance{
			Address: c.Etcd.Address, GRPCPort: fmt.Sprint(c.Port), HTTPPort: fmt.Sprint(c.Etcd.HTTPPort),
		},
	}
}

func debugLog(message string, args ...any) {
	g.Log().Debug(context.Background(), append([]any{message}, args...)...)
}

func errorLog(message string, args ...any) {
	g.Log().Error(context.Background(), append([]any{message}, args...)...)
}
