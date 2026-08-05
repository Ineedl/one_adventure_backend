package computing

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gogf/gf/v2/container/gvar"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	computingpb "one_adventure_rpc/proto/computing"
	"one_adventure_servicekit/registration"
)

func TestConfigValidate(t *testing.T) {
	for _, port := range []int{1, 8082, 65535} {
		if err := (Config{Port: port, Registration: validRegistrationConfig()}).validate(); err != nil {
			t.Fatalf("Config{Port: %d}.validate() error = %v", port, err)
		}
	}
	for _, port := range []int{0, -1, 65536} {
		if err := (Config{Port: port, Registration: validRegistrationConfig()}).validate(); err == nil {
			t.Fatalf("Config{Port: %d}.validate() error = nil", port)
		}
	}
	if err := (Config{Port: 8082}).validate(); err == nil {
		t.Fatal("Config without registration.validate() error = nil")
	}
}

func validRegistrationConfig() RegistrationConfig {
	return RegistrationConfig{
		GatewayIP:   "127.0.0.1",
		GatewayPort: 8081,
		ServiceType: "computing",
		ServiceIP:   "127.0.0.1",
		InstanceID:  "computing-1",
	}
}

func TestConfigScanRegistration(t *testing.T) {
	var cfg Config
	err := gvar.New(map[string]any{
		"port": 8082,
		"registration": map[string]any{
			"gatewayIp":          "127.0.0.1",
			"gatewayPort":        8081,
			"serviceType":        "computing",
			"serviceIp":          "127.0.0.1",
			"instanceId":         "",
			"registerTimeout":    "8s",
			"gatewayPingTimeout": "30s",
		},
	}).Scan(&cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if err = cfg.validate(); err != nil {
		t.Fatalf("Config.validate() error = %v", err)
	}
	if cfg.Registration.RegisterTimeout != 8*time.Second ||
		cfg.Registration.GatewayPingTimeout != 30*time.Second {
		t.Fatalf("scanned registration durations = %+v", cfg.Registration)
	}
}

func TestComputingRPCServer(t *testing.T) {
	server := newServer(Config{Port: 8082}, newComputingService(), nil)
	listener := bufconn.Listen(1024 * 1024)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.grpcServer.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	clientConn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = clientConn.Close() })
	client := computingpb.NewComputingServiceClient(clientConn)

	response, err := client.CollisionCalculation(context.Background(), &computingpb.CollisionCalculationReq{})
	if err != nil {
		t.Fatalf("CollisionCalculation() error = %v", err)
	}
	if response == nil {
		t.Fatal("CollisionCalculation() response = nil")
	}

	server.Stop()
	if err = <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		t.Fatalf("Serve() error = %v", err)
	}
}

func TestNewServerRegistersSharedPingService(t *testing.T) {
	cfg := Config{Port: 8082, Registration: validRegistrationConfig()}
	manager, err := registration.New(cfg.registrationConfig())
	if err != nil {
		t.Fatalf("registration.New() error = %v", err)
	}
	server := newServer(cfg, newComputingService(), manager)
	services := server.grpcServer.GetServiceInfo()
	if _, ok := services["computing.ComputingService"]; !ok {
		t.Fatal("computing.ComputingService is not registered")
	}
	if _, ok := services["ping.PingService"]; !ok {
		t.Fatal("ping.PingService is not registered")
	}
}
