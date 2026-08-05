package registration

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	gatewaypb "one_adventure_rpc/proto/gateway"
	pingpb "one_adventure_rpc/proto/ping"
)

type testGatewayService struct {
	gatewaypb.UnimplementedGatewayServiceServer
	onRegister func(context.Context, *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error)
}

func (s *testGatewayService) RegisterGateway(ctx context.Context, request *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
	return s.onRegister(ctx, request)
}

func TestManagerBecomesActiveAfterSuccessfulRegistration(t *testing.T) {
	gatewayListener := listenLocal(t)
	serviceListener := listenLocal(t)
	gatewayHost, gatewayPort := listenerAddress(t, gatewayListener)
	serviceHost, servicePort := listenerAddress(t, serviceListener)

	registered := make(chan *gatewaypb.RegisterReq, 1)
	gatewayService := &testGatewayService{onRegister: func(ctx context.Context, request *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
		if err := pingRegisteredService(ctx, request); err != nil {
			return &gatewaypb.RegisterResp{Code: 502, Message: err.Error()}, nil
		}
		registered <- request
		return &gatewaypb.RegisterResp{Code: 0, Message: "success", PingIntervalMs: 10, LeaseDurationMs: 200}, nil
	}}
	startGatewayServer(t, gatewayListener, gatewayService)

	manager := newTestManager(t, gatewayHost, gatewayPort, serviceHost, servicePort)
	startPingServer(t, serviceListener, manager)
	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- manager.Run(runCtx) }()

	select {
	case request := <-registered:
		if request.GetType() != "computing" || request.GetInstanceId() != "computing-1" || request.GetRegistrationToken() == "" {
			t.Fatalf("RegisterReq = %#v", request)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for registration")
	}
	waitForState(t, manager, StateActive)

	cancel()
	if err := <-runDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("Manager.Run() error = %v", err)
	}
}

func TestManagerDoesNotWaitForInitialPingAfterRegistration(t *testing.T) {
	gatewayListener := listenLocal(t)
	serviceListener := listenLocal(t)
	gatewayHost, gatewayPort := listenerAddress(t, gatewayListener)
	serviceHost, servicePort := listenerAddress(t, serviceListener)

	gatewayService := &testGatewayService{onRegister: func(_ context.Context, _ *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
		return &gatewaypb.RegisterResp{Code: 0, Message: "success"}, nil
	}}
	startGatewayServer(t, gatewayListener, gatewayService)

	manager := newTestManager(t, gatewayHost, gatewayPort, serviceHost, servicePort)
	manager.config.GatewayPingTimeout = 200 * time.Millisecond
	startPingServer(t, serviceListener, manager)
	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- manager.Run(runCtx) }()

	waitForState(t, manager, StateActive)
	cancel()
	if err := <-runDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("Manager.Run() error = %v", err)
	}
}

func TestManagerReregistersWhenGatewayPingsStop(t *testing.T) {
	gatewayListener := listenLocal(t)
	serviceListener := listenLocal(t)
	gatewayHost, gatewayPort := listenerAddress(t, gatewayListener)
	serviceHost, servicePort := listenerAddress(t, serviceListener)

	var registrations atomic.Int32
	gatewayService := &testGatewayService{onRegister: func(ctx context.Context, request *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
		registrations.Add(1)
		if err := pingRegisteredService(ctx, request); err != nil {
			return &gatewaypb.RegisterResp{Code: 502, Message: err.Error()}, nil
		}
		return &gatewaypb.RegisterResp{Code: 0, Message: "success", PingIntervalMs: 5, LeaseDurationMs: 10_000}, nil
	}}
	startGatewayServer(t, gatewayListener, gatewayService)

	manager := newTestManager(t, gatewayHost, gatewayPort, serviceHost, servicePort)
	manager.config.GatewayPingTimeout = 20 * time.Millisecond
	startPingServer(t, serviceListener, manager)
	runCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err := manager.Run(runCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Manager.Run() error = %v", err)
	}
	if count := registrations.Load(); count < 2 {
		t.Fatalf("registration attempts = %d, want at least 2", count)
	}
}

func TestManagerDoesNotApplyGatewayPingTimeoutWhileRegistering(t *testing.T) {
	gatewayListener := listenLocal(t)
	serviceListener := listenLocal(t)
	gatewayHost, gatewayPort := listenerAddress(t, gatewayListener)
	serviceHost, servicePort := listenerAddress(t, serviceListener)

	var registrations atomic.Int32
	gatewayService := &testGatewayService{onRegister: func(ctx context.Context, _ *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
		registrations.Add(1)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	startGatewayServer(t, gatewayListener, gatewayService)

	manager := newTestManager(t, gatewayHost, gatewayPort, serviceHost, servicePort)
	manager.config.GatewayPingTimeout = 10 * time.Millisecond
	startPingServer(t, serviceListener, manager)
	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- manager.Run(runCtx) }()

	deadline := time.Now().Add(time.Second)
	for registrations.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if registrations.Load() == 0 {
		t.Fatal("registration RPC was not started")
	}
	time.Sleep(40 * time.Millisecond)
	if count := registrations.Load(); count != 1 {
		t.Fatalf("registration attempts = %d, want 1 while registration is still running", count)
	}
	if state := manager.State(); state != StateRegistering {
		t.Fatalf("Manager.State() = %s, want %s", state, StateRegistering)
	}

	cancel()
	if err := <-runDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("Manager.Run() error = %v", err)
	}
}

func TestManagerRemainsActiveWhileGatewayPings(t *testing.T) {
	gatewayListener := listenLocal(t)
	serviceListener := listenLocal(t)
	gatewayHost, gatewayPort := listenerAddress(t, gatewayListener)
	serviceHost, servicePort := listenerAddress(t, serviceListener)

	var registrations atomic.Int32
	stopPings := make(chan struct{})
	gatewayService := &testGatewayService{onRegister: func(ctx context.Context, request *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
		registrations.Add(1)
		connection, client, err := newPingClient(request)
		if err != nil {
			return nil, err
		}
		if _, err = client.Ping(ctx, pingRequest(request)); err != nil {
			_ = connection.Close()
			return nil, err
		}
		go func() {
			defer connection.Close()
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					pingCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
					_, _ = client.Ping(pingCtx, pingRequest(request))
					cancel()
				case <-stopPings:
					return
				}
			}
		}()
		return &gatewaypb.RegisterResp{Code: 0, Message: "success", PingIntervalMs: 5, LeaseDurationMs: 25}, nil
	}}
	startGatewayServer(t, gatewayListener, gatewayService)

	manager := newTestManager(t, gatewayHost, gatewayPort, serviceHost, servicePort)
	startPingServer(t, serviceListener, manager)
	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- manager.Run(runCtx) }()
	waitForState(t, manager, StateActive)
	time.Sleep(60 * time.Millisecond)
	if count := registrations.Load(); count != 1 {
		t.Fatalf("registration attempts = %d, want 1 while pings continue", count)
	}
	close(stopPings)
	cancel()
	if err := <-runDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("Manager.Run() error = %v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	manager, err := New(Config{
		GatewayIP:   "127.0.0.1",
		GatewayPort: 8081,
		Service:     ServiceInfo{Type: "computing", IP: "127.0.0.1", Port: 8082, InstanceID: "computing-1"},
	}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if manager.config.RegisterTimeout != defaultRegisterTimeout ||
		manager.config.GatewayPingTimeout != defaultGatewayPingTimeout {
		t.Fatalf("default config = %+v", manager.config)
	}
}

func TestEmptyInstanceIDUsesServiceNameAndMachineCode(t *testing.T) {
	config := Config{
		GatewayIP:   "127.0.0.1",
		GatewayPort: 8081,
		Service:     ServiceInfo{Type: "Computing Service", IP: "127.0.0.1", Port: 8082},
	}
	first, err := New(config, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New(first) error = %v", err)
	}
	second, err := New(config, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New(second) error = %v", err)
	}
	if !strings.HasPrefix(first.config.Service.InstanceID, "computing_service_") {
		t.Fatalf("generated instance id = %q", first.config.Service.InstanceID)
	}
	if first.config.Service.InstanceID != second.config.Service.InstanceID {
		t.Fatalf("generated instance ids differ: %q != %q", first.config.Service.InstanceID, second.config.Service.InstanceID)
	}
}

func newTestManager(t *testing.T, gatewayHost string, gatewayPort int, serviceHost string, servicePort int) *Manager {
	t.Helper()
	manager, err := New(Config{
		GatewayIP: gatewayHost, GatewayPort: gatewayPort,
		Service:              ServiceInfo{Type: "computing", IP: serviceHost, Port: servicePort, Version: "v1.0.0", InstanceID: "computing-1"},
		RegisterTimeout:      time.Second,
		GatewayPingTimeout:   100 * time.Millisecond,
		RetryInitialInterval: 5 * time.Millisecond, RetryMaxInterval: 10 * time.Millisecond,
	}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return manager
}

func pingRegisteredService(ctx context.Context, request *gatewaypb.RegisterReq) error {
	connection, client, err := newPingClient(request)
	if err != nil {
		return err
	}
	defer connection.Close()
	response, err := client.Ping(ctx, pingRequest(request))
	if err != nil {
		return err
	}
	if response.GetCode() != 0 {
		return errors.New(response.GetMessage())
	}
	return nil
}

func newPingClient(request *gatewaypb.RegisterReq) (*grpc.ClientConn, pingpb.PingServiceClient, error) {
	connection, err := grpc.NewClient(
		net.JoinHostPort(request.GetIp(), strconv.Itoa(int(request.GetPort()))),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}
	return connection, pingpb.NewPingServiceClient(connection), nil
}

func pingRequest(request *gatewaypb.RegisterReq) *pingpb.PingReq {
	return &pingpb.PingReq{Type: request.GetType(), InstanceId: request.GetInstanceId(), RegistrationToken: request.GetRegistrationToken()}
}

func waitForState(t *testing.T, manager *Manager, wanted State) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if manager.State() == wanted {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("Manager.State() = %s, want %s", manager.State(), wanted)
}

func listenLocal(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	return listener
}

func listenerAddress(t *testing.T, listener net.Listener) (string, int) {
	t.Helper()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("net.SplitHostPort() error = %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("strconv.Atoi() error = %v", err)
	}
	return host, port
}

func startGatewayServer(t *testing.T, listener net.Listener, service gatewaypb.GatewayServiceServer) {
	t.Helper()
	server := grpc.NewServer()
	gatewaypb.RegisterGatewayServiceServer(server, service)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
}

func startPingServer(t *testing.T, listener net.Listener, manager *Manager) {
	t.Helper()
	server := grpc.NewServer()
	manager.RegisterPingService(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
