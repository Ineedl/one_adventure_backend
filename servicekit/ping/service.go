// Package ping provides the shared PingService implementation that every
// microservice exposes to the gateway.
package ping

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
	pingpb "one_adventure_rpc/proto/ping"
)

const (
	responseCodeSuccess  = int32(0)
	responseCodeInvalid  = int32(400)
	responseCodeNotFound = int32(404)
)

// Service implements the common PingService and validates that probes belong
// to the currently active registration attempt.
type Service struct {
	pingpb.UnimplementedPingServiceServer

	serviceType string
	instanceID  string

	mu                sync.RWMutex
	registrationToken string
	activity          chan struct{}
}

func New(serviceType, instanceID string) (*Service, error) {
	serviceType = strings.ToLower(strings.TrimSpace(serviceType))
	instanceID = strings.TrimSpace(instanceID)
	if serviceType == "" {
		return nil, fmt.Errorf("service type is required")
	}
	if instanceID == "" {
		return nil, fmt.Errorf("service instance id is required")
	}
	return &Service{
		serviceType: serviceType,
		instanceID:  instanceID,
		activity:    make(chan struct{}, 1),
	}, nil
}

// Register installs the common PingService on a microservice gRPC server.
func (s *Service) Register(registrar grpc.ServiceRegistrar) {
	pingpb.RegisterPingServiceServer(registrar, s)
}

// Activate switches Ping validation to a new registration token and discards
// activity left by the previous registration attempt.
func (s *Service) Activate(registrationToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		select {
		case <-s.activity:
		default:
			s.registrationToken = strings.TrimSpace(registrationToken)
			return
		}
	}
}

// Deactivate clears the token only when it still belongs to the supplied
// registration attempt.
func (s *Service) Deactivate(registrationToken string) {
	s.mu.Lock()
	if s.registrationToken == registrationToken {
		s.registrationToken = ""
	}
	s.mu.Unlock()
}

// Activity reports valid gateway probes. The channel coalesces bursts because
// registration only needs to know whether at least one recent Ping arrived.
func (s *Service) Activity() <-chan struct{} {
	return s.activity
}

func (s *Service) Ping(_ context.Context, request *pingpb.PingReq) (*pingpb.PingResp, error) {
	if request == nil {
		return &pingpb.PingResp{Code: responseCodeInvalid, Message: "ping request is required"}, nil
	}
	if strings.ToLower(strings.TrimSpace(request.GetType())) != s.serviceType ||
		strings.TrimSpace(request.GetInstanceId()) != s.instanceID {
		return &pingpb.PingResp{Code: responseCodeNotFound, Message: "service instance not found"}, nil
	}

	s.mu.RLock()
	valid := s.registrationToken != "" && request.GetRegistrationToken() == s.registrationToken
	if valid {
		select {
		case s.activity <- struct{}{}:
		default:
		}
	}
	s.mu.RUnlock()
	if !valid {
		return &pingpb.PingResp{Code: responseCodeNotFound, Message: "registration attempt not found"}, nil
	}
	return &pingpb.PingResp{Code: responseCodeSuccess, Message: "success"}, nil
}
