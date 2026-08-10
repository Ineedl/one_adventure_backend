package service

import (
	"errors"

	"google.golang.org/grpc"
)

var ErrServiceUnavailable = errors.New("service is unavailable")

// ServiceResolver resolves the current gRPC connection for a registered
// microservice. Resolution happens for every HTTP request so reconnects and
// health-check removals are reflected immediately.
type ServiceResolver interface {
	ResolveService(serviceName string) (grpc.ClientConnInterface, error)
}
