package discovery

import (
	"reflect"
	"testing"
	"time"
)

func TestConfigValidateRequiresPositiveTimeout(t *testing.T) {
	config := Config{Endpoints: []string{"127.0.0.1:2379"}}
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want a positive timeout validation error")
	}
	config.DialTimeout = time.Second
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestNewDiscovererUsesEnvoy(t *testing.T) {
	t.Setenv("ENVOY_ADDRESS", "")
	discoverer, err := NewDiscoverer(Config{})
	if err != nil {
		t.Fatalf("NewDiscoverer() error = %v", err)
	}
	defer discoverer.Close()
	if discoverer.envoyAddress != DefaultEnvoyAddress {
		t.Fatalf("envoy address = %q, want %q", discoverer.envoyAddress, DefaultEnvoyAddress)
	}
	if discoverer.client != nil {
		t.Fatal("discoverer unexpectedly created an etcd client")
	}
}

func TestWatchPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		watchAll bool
		want     []string
	}{
		{name: "microservice watches nothing", services: []string{}, want: []string{}},
		{name: "microservice watches configured services", services: []string{"user", "computing"}, want: []string{"/one_adventure/computing/", "/one_adventure/user/"}},
		{name: "gateway watches all", watchAll: true, want: []string{"/one_adventure/"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := watchPrefixes(test.services, test.watchAll); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("watchPrefixes() = %#v, want %#v", got, test.want)
			}
		})
	}
}
