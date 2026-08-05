package gateway

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	gatewaypb "one_adventure_rpc/proto/gateway"
	pingpb "one_adventure_rpc/proto/ping"
)

const (
	successMessage      = "success"
	responseCodeSuccess = int32(0)
	responseCodeInvalid = int32(400)
	responseCodeProbe   = int32(502)
	missedPingLimit     = 3
)

// gatewayService validates and tracks live microservice registrations.
type gatewayService struct {
	gatewaypb.UnimplementedGatewayServiceServer

	registry     *serviceRegistry
	pingInterval time.Duration
	pingTimeout  time.Duration

	lifecycleMu sync.Mutex
	pingCancel  context.CancelFunc
	closed      bool
}

func newGatewayService(cfg Config) *gatewayService {
	return &gatewayService{
		registry:     newServiceRegistry(),
		pingInterval: cfg.PingInterval,
		pingTimeout:  cfg.PingTimeout,
	}
}

func (s *gatewayService) RegisterGateway(ctx context.Context, request *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
	serviceType, err := validateRegistration(request)
	if err != nil {
		return &gatewaypb.RegisterResp{Code: responseCodeInvalid, Message: err.Error()}, nil
	}

	connection, err := grpc.NewClient(
		net.JoinHostPort(strings.TrimSpace(request.GetIp()), strconv.Itoa(int(request.GetPort()))),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return &gatewaypb.RegisterResp{Code: responseCodeProbe, Message: fmt.Sprintf("create grpc client: %v", err)}, nil
	}

	client := pingpb.NewPingServiceClient(connection)
	if err = s.ping(ctx, client, request); err != nil {
		_ = connection.Close()
		return &gatewaypb.RegisterResp{Code: responseCodeProbe, Message: fmt.Sprintf("ping %s instance: %v", serviceType, err)}, nil
	}

	key := instanceKey{serviceType: serviceType, instanceID: strings.TrimSpace(request.GetInstanceId())}
	if err = s.registry.replace(key, request, connection, client, time.Now()); err != nil {
		_ = connection.Close()
		return &gatewaypb.RegisterResp{Code: responseCodeProbe, Message: err.Error()}, nil
	}

	g.Log().Infof(ctx, "registered %s instance %s at %s:%d", serviceType, key.instanceID, request.GetIp(), request.GetPort())
	leaseDuration := missedPingLimit*s.pingInterval + s.pingTimeout
	return &gatewaypb.RegisterResp{
		Code:            responseCodeSuccess,
		Message:         successMessage,
		PingIntervalMs:  s.pingInterval.Milliseconds(),
		LeaseDurationMs: leaseDuration.Milliseconds(),
	}, nil
}

func (s *gatewayService) ping(ctx context.Context, client pingpb.PingServiceClient, registration *gatewaypb.RegisterReq) error {
	pingCtx, cancel := context.WithTimeout(ctx, s.pingTimeout)
	defer cancel()
	response, err := client.Ping(pingCtx, &pingpb.PingReq{
		Type:              registration.GetType(),
		InstanceId:        registration.GetInstanceId(),
		RegistrationToken: registration.GetRegistrationToken(),
	})
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("empty response")
	}
	if response.GetCode() != responseCodeSuccess {
		return fmt.Errorf("rejected with code %d: %s", response.GetCode(), response.GetMessage())
	}
	return nil
}

func validateRegistration(request *gatewaypb.RegisterReq) (string, error) {
	if request == nil {
		return "", fmt.Errorf("registration is required")
	}
	serviceType := strings.ToLower(strings.TrimSpace(request.GetType()))
	if serviceType == "" {
		return "", fmt.Errorf("service type is required")
	}
	if net.ParseIP(strings.TrimSpace(request.GetIp())) == nil {
		return "", fmt.Errorf("a valid ip is required")
	}
	if request.GetPort() < 1 || request.GetPort() > 65535 {
		return "", fmt.Errorf("port must be between 1 and 65535")
	}
	if strings.TrimSpace(request.GetInstanceId()) == "" {
		return "", fmt.Errorf("instance_id is required")
	}
	if strings.TrimSpace(request.GetRegistrationToken()) == "" {
		return "", fmt.Errorf("registration_token is required")
	}
	return serviceType, nil
}

func (s *gatewayService) startHealthChecks() {
	s.lifecycleMu.Lock()
	if s.closed || s.pingCancel != nil {
		s.lifecycleMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.pingCancel = cancel
	s.lifecycleMu.Unlock()

	go func() {
		ticker := time.NewTicker(s.pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.probeRegisteredInstances(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *gatewayService) probeRegisteredInstances(ctx context.Context) {
	targets := s.registry.probeTargets()
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(targets))
	for _, target := range targets {
		target := target
		go func() {
			defer waitGroup.Done()
			if err := s.ping(ctx, target.client, target.registration); err != nil {
				if s.registry.remove(target.key, target.registration.GetRegistrationToken()) {
					g.Log().Warningf(context.Background(), "removed unreachable %s instance %s after ping failed: %v", target.key.serviceType, target.key.instanceID, err)
				}
				return
			}
			s.registry.markPing(target.key, target.registration.GetRegistrationToken(), time.Now())
		}()
	}
	waitGroup.Wait()
}

func (s *gatewayService) close() {
	s.lifecycleMu.Lock()
	if s.closed {
		s.lifecycleMu.Unlock()
		return
	}
	s.closed = true
	cancel := s.pingCancel
	s.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.registry.close()
}
