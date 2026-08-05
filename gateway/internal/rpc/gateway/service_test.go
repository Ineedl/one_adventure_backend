package gateway

import (
	"context"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gogf/gf/v2/container/gvar"
	"google.golang.org/grpc"
	gatewaypb "one_adventure_rpc/proto/gateway"
	pingpb "one_adventure_rpc/proto/ping"
)

type testPingService struct {
	pingpb.UnimplementedPingServiceServer
	code  atomic.Int32
	count atomic.Int32
}

func (s *testPingService) Ping(_ context.Context, _ *pingpb.PingReq) (*pingpb.PingResp, error) {
	s.count.Add(1)
	return &pingpb.PingResp{Code: s.code.Load(), Message: "ping"}, nil
}

func TestConfigValidate(t *testing.T) {
	valid := []Config{
		{Port: 1, PingInterval: time.Millisecond, PingTimeout: time.Millisecond},
		{Port: 8081, PingInterval: 10 * time.Second, PingTimeout: 3 * time.Second},
		{Port: 65535, PingInterval: time.Hour, PingTimeout: time.Minute},
	}
	for _, cfg := range valid {
		if err := cfg.validate(); err != nil {
			t.Fatalf("Config%+v.validate() error = %v", cfg, err)
		}
	}

	invalid := []Config{
		{Port: 0, PingInterval: time.Second, PingTimeout: time.Second},
		{Port: 65536, PingInterval: time.Second, PingTimeout: time.Second},
		{Port: 8081, PingInterval: 0, PingTimeout: time.Second},
		{Port: 8081, PingInterval: time.Second, PingTimeout: 0},
	}
	for _, cfg := range invalid {
		if err := cfg.validate(); err == nil {
			t.Fatalf("Config%+v.validate() error = nil", cfg)
		}
	}
}

func TestConfigScanPingSettings(t *testing.T) {
	var cfg Config
	err := gvar.New(map[string]any{
		"port":         8081,
		"pingInterval": "10s",
		"pingTimeout":  "3s",
	}).Scan(&cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if cfg.Port != 8081 || cfg.PingInterval != 10*time.Second || cfg.PingTimeout != 3*time.Second {
		t.Fatalf("scanned Config = %+v", cfg)
	}
}

func TestRegisterGatewayProbesAndStoresMultipleInstances(t *testing.T) {
	host, port, pingService := startTestMicroservice(t)
	service := newGatewayService(Config{PingInterval: 10 * time.Second, PingTimeout: time.Second})
	t.Cleanup(service.close)

	first := &gatewaypb.RegisterReq{
		Type: "computing", Ip: host, Port: port, RegisterTime: 123456789,
		Version: "v1.2.3", Weight: 10, InstanceId: "computing-1", RegistrationToken: "token-1",
	}
	firstResponse := assertRegistrationSucceeded(t, service, first)
	wantLease := missedPingLimit*service.pingInterval + service.pingTimeout
	if firstResponse.GetPingIntervalMs() != service.pingInterval.Milliseconds() || firstResponse.GetLeaseDurationMs() != wantLease.Milliseconds() {
		t.Fatalf("RegisterGateway() ping response = %#v", firstResponse)
	}
	assertRegistrationSucceeded(t, service, &gatewaypb.RegisterReq{
		Type: "computing", Ip: host, Port: port, Version: "v1.2.4", Weight: 20,
		InstanceId: "computing-2", RegistrationToken: "token-2",
	})

	if count := service.registry.count("computing"); count != 2 {
		t.Fatalf("registered computing instances = %d, want 2", count)
	}
	if count := pingService.count.Load(); count != 2 {
		t.Fatalf("initial Ping calls = %d, want 2", count)
	}
	stored, lastPing, ok := service.registry.instance(instanceKey{serviceType: "computing", instanceID: "computing-1"})
	if !ok || lastPing.IsZero() {
		t.Fatal("computing-1 was not stored with its initial Ping time")
	}
	if stored.GetType() != first.GetType() || stored.GetIp() != first.GetIp() || stored.GetPort() != first.GetPort() ||
		stored.GetRegisterTime() != first.GetRegisterTime() || stored.GetVersion() != first.GetVersion() ||
		stored.GetWeight() != first.GetWeight() || stored.GetInstanceId() != first.GetInstanceId() ||
		stored.GetRegistrationToken() != first.GetRegistrationToken() {
		t.Fatalf("stored registration = %#v, want %#v", stored, first)
	}
}

func TestGatewayPeriodicProbeRefreshesAndRemovesInstances(t *testing.T) {
	host, port, pingService := startTestMicroservice(t)
	service := newGatewayService(Config{PingInterval: time.Second, PingTimeout: time.Second})
	t.Cleanup(service.close)
	request := &gatewaypb.RegisterReq{
		Type: "computing", Ip: host, Port: port, InstanceId: "computing-1", RegistrationToken: "token-1",
	}
	assertRegistrationSucceeded(t, service, request)
	key := instanceKey{serviceType: "computing", instanceID: "computing-1"}
	_, initialPing, _ := service.registry.instance(key)

	time.Sleep(time.Millisecond)
	service.probeRegisteredInstances(context.Background())
	_, refreshedPing, ok := service.registry.instance(key)
	if !ok || !refreshedPing.After(initialPing) {
		t.Fatalf("last Ping = %v, want after %v", refreshedPing, initialPing)
	}

	pingService.code.Store(1)
	service.probeRegisteredInstances(context.Background())
	if count := service.registry.count("computing"); count != 0 {
		t.Fatalf("registered computing instances = %d, want 0 after failed Ping", count)
	}
}

func TestRegisterGatewayRejectsFailedPing(t *testing.T) {
	host, port, pingService := startTestMicroservice(t)
	pingService.code.Store(1)
	service := newGatewayService(Config{PingInterval: time.Second, PingTimeout: time.Second})
	t.Cleanup(service.close)

	response, err := service.RegisterGateway(context.Background(), &gatewaypb.RegisterReq{
		Type: "computing", Ip: host, Port: port, InstanceId: "computing-failed", RegistrationToken: "failed-token",
	})
	if err != nil {
		t.Fatalf("RegisterGateway() error = %v", err)
	}
	if response.Code != responseCodeProbe {
		t.Fatalf("RegisterGateway() response = %#v, want probe failure", response)
	}
	if count := service.registry.count("computing"); count != 0 {
		t.Fatalf("registered computing instances = %d, want 0", count)
	}
}

func TestRegistryIgnoresRemovalFromStaleRegistration(t *testing.T) {
	registry := newServiceRegistry()
	t.Cleanup(registry.close)
	key := instanceKey{serviceType: "computing", instanceID: "computing-1"}
	if err := registry.replace(key, &gatewaypb.RegisterReq{Type: "computing", InstanceId: key.instanceID, RegistrationToken: "token-1"}, nil, nil, time.Now()); err != nil {
		t.Fatalf("replace(first) error = %v", err)
	}
	if err := registry.replace(key, &gatewaypb.RegisterReq{Type: "computing", InstanceId: key.instanceID, RegistrationToken: "token-2"}, nil, nil, time.Now()); err != nil {
		t.Fatalf("replace(second) error = %v", err)
	}
	if registry.remove(key, "token-1") {
		t.Fatal("stale registration removed the replacement instance")
	}
	stored, _, ok := registry.instance(key)
	if !ok || stored.GetRegistrationToken() != "token-2" {
		t.Fatalf("stored registration = %#v", stored)
	}
}

func assertRegistrationSucceeded(t *testing.T, service *gatewayService, request *gatewaypb.RegisterReq) *gatewaypb.RegisterResp {
	t.Helper()
	response, err := service.RegisterGateway(context.Background(), request)
	if err != nil {
		t.Fatalf("RegisterGateway(%s) error = %v", request.GetInstanceId(), err)
	}
	if response.Code != responseCodeSuccess || response.Message != successMessage {
		t.Fatalf("RegisterGateway(%s) response = %#v", request.GetInstanceId(), response)
	}
	return response
}

func startTestMicroservice(t *testing.T) (string, int32, *testPingService) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	pingService := &testPingService{}
	server := grpc.NewServer()
	pingpb.RegisterPingServiceServer(server, pingService)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("net.SplitHostPort() error = %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("strconv.Atoi() error = %v", err)
	}
	return host, int32(port), pingService
}
