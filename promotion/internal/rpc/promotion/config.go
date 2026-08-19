package promotion

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/os/gcfg"
	"one_adventure_servicekit/discovery"
)

type Config struct {
	Port         int        `json:"port"`
	EnvoyAddress string     `json:"envoyAddress"`
	Etcd         EtcdConfig `json:"etcd"`
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
	a, err := gcfg.NewAdapterFile("rpc.yaml")
	if err != nil {
		return Config{}, err
	}
	v, err := gcfg.NewWithAdapter(a).Get(ctx, ".")
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err = v.Scan(&c); err != nil {
		return Config{}, err
	}
	if c.Port < 1 || c.Port > 65535 {
		return Config{}, fmt.Errorf("invalid rpc port")
	}
	if err = c.discoveryConfig().Validate(); err != nil {
		return Config{}, err
	}
	return c, c.registration().Validate()
}
func (c Config) discoveryConfig() discovery.Config {
	return discovery.Config{Endpoints: c.Etcd.Endpoints, DialTimeout: c.Etcd.DialTimeout, EnvoyAddress: c.EnvoyAddress}
}
func (c Config) registration() discovery.Registration {
	return discovery.Registration{ServerName: c.Etcd.ServerName, InstanceID: c.Etcd.InstanceID, LeaseTTL: c.Etcd.LeaseTTL, Instance: discovery.Instance{Address: c.Etcd.Address, GRPCPort: fmt.Sprint(c.Port), HTTPPort: fmt.Sprint(c.Etcd.HTTPPort)}}
}
