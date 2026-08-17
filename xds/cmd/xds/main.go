package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
	"one_adventure_xds"
)

type fileConfig struct {
	Etcd struct {
		Endpoints   []string      `yaml:"endpoints"`
		DialTimeout time.Duration `yaml:"dialTimeout"`
	} `yaml:"etcd"`
	Listen struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"listen"`
	Services []string `yaml:"services"`
}

func main() {
	// 接收终止信号并将取消信号传递给 xDS 服务，实现优雅退出。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := loadConfig(getenv("XDS_CONFIG", "config.yaml"))
	if err != nil {
		log.Fatal(err)
	}
	if endpoints := os.Getenv("ETCD_ENDPOINTS"); endpoints != "" {
		cfg.Etcd.Endpoints = strings.Split(endpoints, ",")
	}
	if err := xds.Run(ctx, xds.Config{Endpoints: cfg.Etcd.Endpoints, DialTimeout: cfg.Etcd.DialTimeout, ListenAddress: cfg.Listen.Address, ListenPort: cfg.Listen.Port, Services: cfg.Services}); err != nil {
		log.Fatal(err)
	}
}

// loadConfig 从 YAML 文件读取 xDS 控制面配置。
func loadConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fileConfig{}, fmt.Errorf("read xds config: %w", err)
	}
	var cfg fileConfig
	// yaml.Unmarshal 将配置文件映射到带 yaml tag 的结构体。
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse xds config: %w", err)
	}
	return cfg, nil
}

// getenv 读取环境变量；变量不存在或为空时返回默认值。
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
