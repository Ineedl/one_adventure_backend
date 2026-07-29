package gateway

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestConfigValidate(t *testing.T) {
	for _, port := range []int{1, 8081, 65535} {
		if err := (Config{Port: port}).validate(); err != nil {
			t.Fatalf("Config{Port: %d}.validate() error = %v", port, err)
		}
	}
	for _, port := range []int{0, -1, 65536} {
		if err := (Config{Port: port}).validate(); err == nil {
			t.Fatalf("Config{Port: %d}.validate() error = nil", port)
		}
	}
}

func TestGatewayRPCServer(t *testing.T) {
	server := newServer(Config{Port: 8081}, newGatewayService())
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
	client := NewGatewayServiceClient(clientConn)

	registerResponse, err := client.RegisterGateway(context.Background(), &RegisterReq{InstanceId: "gateway-1"})
	if err != nil {
		t.Fatalf("RegisterGateway() error = %v", err)
	}
	if registerResponse.Code != 0 || registerResponse.Message != successMessage {
		t.Fatalf("RegisterGateway() response = %#v", registerResponse)
	}

	heartResponse, err := client.Heart(context.Background(), &HeartReq{})
	if err != nil {
		t.Fatalf("Heart() error = %v", err)
	}
	if heartResponse.Code != 0 || heartResponse.Message != successMessage {
		t.Fatalf("Heart() response = %#v", heartResponse)
	}

	server.Stop()
	if err = <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		t.Fatalf("Serve() error = %v", err)
	}
}
