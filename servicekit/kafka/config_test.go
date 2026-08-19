package kafka

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigReadsKafkaYAML(t *testing.T) {
	directory := t.TempDir()
	config := []byte("brokers: [\"kafka:19092\"]\ngroup: \"promotion\"\nwriteTimeout: \"3s\"\n")
	if err := os.WriteFile(filepath.Join(directory, "kafka.yaml"), config, 0o600); err != nil {
		t.Fatalf("write kafka config: %v", err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err = os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(workingDirectory); chdirErr != nil {
			t.Errorf("restore working directory: %v", chdirErr)
		}
	})

	c, err := LoadConfig(context.Background())
	if err != nil {
		t.Fatalf("load kafka config: %v", err)
	}
	if len(c.Brokers) != 1 || c.Brokers[0] != "kafka:19092" {
		t.Fatalf("unexpected brokers: %v", c.Brokers)
	}
	if c.Group != "promotion" || c.WriteTimeout != 3*time.Second {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestConfigWithDefaults(t *testing.T) {
	c := (Config{}).WithDefaults()
	if len(c.Brokers) != 1 || c.Brokers[0] != "127.0.0.1:9092" {
		t.Fatalf("unexpected brokers: %v", c.Brokers)
	}
	if c.Group != "one-adventure" {
		t.Fatalf("unexpected group: %q", c.Group)
	}
	if c.MaxWait != time.Second || c.ReadTimeout != 10*time.Second || c.WriteTimeout != 10*time.Second {
		t.Fatalf("unexpected timeouts: %+v", c)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestConfigRejectsEmptyBroker(t *testing.T) {
	if err := (Config{Brokers: []string{" "}}).Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
