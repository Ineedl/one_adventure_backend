package computing

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"one_adventure_servicekit/registration"
)

// Config contains the computing gRPC server configuration stored under the
// "rpc.computing" node.
type Config struct {
	Port         int                `json:"port"`
	Registration RegistrationConfig `json:"registration"`
}

type RegistrationConfig struct {
	GatewayIP          string        `json:"gatewayIp"`
	GatewayPort        int           `json:"gatewayPort"`
	ServiceType        string        `json:"serviceType"`
	ServiceIP          string        `json:"serviceIp"`
	Version            string        `json:"version"`
	Weight             int32         `json:"weight"`
	InstanceID         string        `json:"instanceId"`
	RegisterTimeout    time.Duration `json:"registerTimeout"`
	GatewayPingTimeout time.Duration `json:"gatewayPingTimeout"`
	RetryInitial       time.Duration `json:"retryInitialInterval"`
	RetryMax           time.Duration `json:"retryMaxInterval"`
}

func loadConfig(ctx context.Context) (Config, error) {
	value, err := g.Cfg().Get(ctx, "rpc.computing")
	if err != nil {
		return Config{}, fmt.Errorf("read computing rpc config: %w", err)
	}
	if value.IsEmpty() {
		return Config{}, fmt.Errorf("computing rpc config is required")
	}

	var cfg Config
	if err = value.Scan(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse computing rpc config: %w", err)
	}
	if err = cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate computing rpc config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if _, err := registration.New(c.registrationConfig()); err != nil {
		return fmt.Errorf("registration config: %w", err)
	}
	return nil
}

func (c Config) registrationConfig() registration.Config {
	return registration.Config{
		GatewayIP:   c.Registration.GatewayIP,
		GatewayPort: c.Registration.GatewayPort,
		Service: registration.ServiceInfo{
			Type:       c.Registration.ServiceType,
			IP:         c.Registration.ServiceIP,
			Port:       c.Port,
			Version:    c.Registration.Version,
			Weight:     c.Registration.Weight,
			InstanceID: c.Registration.InstanceID,
		},
		RegisterTimeout:      c.Registration.RegisterTimeout,
		GatewayPingTimeout:   c.Registration.GatewayPingTimeout,
		RetryInitialInterval: c.Registration.RetryInitial,
		RetryMaxInterval:     c.Registration.RetryMax,
	}
}
