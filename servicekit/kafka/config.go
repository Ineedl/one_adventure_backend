package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gcfg"
)

// Config describes the Kafka connection and consumer defaults.
type Config struct {
	Brokers      []string      `json:"brokers" yaml:"brokers"`
	Group        string        `json:"group" yaml:"group"`
	MinBytes     int           `json:"minBytes" yaml:"minBytes"`
	MaxBytes     int           `json:"maxBytes" yaml:"maxBytes"`
	MaxWait      time.Duration `json:"maxWait" yaml:"maxWait"`
	ReadTimeout  time.Duration `json:"readTimeout" yaml:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
}

func LoadConfig(ctx context.Context) (Config, error) {
	adapter, err := gcfg.NewAdapterFile("kafka.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("create kafka config adapter: %w", err)
	}
	v, err := gcfg.NewWithAdapter(adapter).Get(ctx, ".")
	if err != nil {
		return Config{}, fmt.Errorf("read kafka config: %w", err)
	}
	if v.IsEmpty() {
		return Config{}, fmt.Errorf("kafka config %q is required", "kafka.yaml")
	}
	var c Config
	if err = v.Scan(&c); err != nil {
		return Config{}, fmt.Errorf("parse kafka config: %w", err)
	}
	c = c.WithDefaults()
	if err = c.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate kafka config: %w", err)
	}
	return c, nil
}

func (c Config) WithDefaults() Config {
	if len(c.Brokers) == 0 {
		c.Brokers = []string{"127.0.0.1:9092"}
	}
	if c.Group == "" {
		c.Group = "one-adventure"
	}
	if c.MinBytes <= 0 {
		c.MinBytes = 1
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = 10e6
	}
	if c.MaxWait <= 0 {
		c.MaxWait = time.Second
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 10 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 10 * time.Second
	}
	return c
}

func (c Config) Validate() error {
	if len(c.Brokers) == 0 {
		return fmt.Errorf("brokers are required")
	}
	for _, broker := range c.Brokers {
		if strings.TrimSpace(broker) == "" {
			return fmt.Errorf("broker must not be empty")
		}
	}
	return nil
}
