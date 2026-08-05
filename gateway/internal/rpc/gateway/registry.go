package gateway

import (
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
	gatewaypb "one_adventure_rpc/proto/gateway"
	pingpb "one_adventure_rpc/proto/ping"
)

var errRegistryClosed = errors.New("service registry is closed")

type instanceKey struct {
	serviceType string
	instanceID  string
}

// registeredInstance owns the registration data and the live connection to
// the registered microservice instance.
type registeredInstance struct {
	registration *gatewaypb.RegisterReq
	connection   *grpc.ClientConn
	pingClient   pingpb.PingServiceClient
	lastPing     time.Time
}

type probeTarget struct {
	key          instanceKey
	registration *gatewaypb.RegisterReq
	client       pingpb.PingServiceClient
}

type serviceRegistry struct {
	mu        sync.RWMutex
	instances map[instanceKey]*registeredInstance
	closed    bool
}

func newServiceRegistry() *serviceRegistry {
	return &serviceRegistry{instances: make(map[instanceKey]*registeredInstance)}
}

func (r *serviceRegistry) replace(key instanceKey, registration *gatewaypb.RegisterReq, connection *grpc.ClientConn, client pingpb.PingServiceClient, now time.Time) error {
	instance := &registeredInstance{
		registration: cloneRegistration(registration),
		connection:   connection,
		pingClient:   client,
		lastPing:     now,
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return errRegistryClosed
	}
	previous := r.instances[key]
	r.instances[key] = instance
	r.mu.Unlock()

	if previous != nil && previous.connection != nil {
		_ = previous.connection.Close()
	}
	return nil
}

func (r *serviceRegistry) markPing(key instanceKey, registrationToken string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	instance, ok := r.instances[key]
	if !ok || r.closed || instance.registration.GetRegistrationToken() != registrationToken {
		return false
	}
	instance.lastPing = now
	return true
}

func (r *serviceRegistry) remove(key instanceKey, registrationToken string) bool {
	r.mu.Lock()
	instance, ok := r.instances[key]
	if !ok || instance.registration.GetRegistrationToken() != registrationToken {
		r.mu.Unlock()
		return false
	}
	delete(r.instances, key)
	r.mu.Unlock()

	if instance.connection != nil {
		_ = instance.connection.Close()
	}
	return true
}

func (r *serviceRegistry) probeTargets() []probeTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return nil
	}
	targets := make([]probeTarget, 0, len(r.instances))
	for key, instance := range r.instances {
		targets = append(targets, probeTarget{
			key:          key,
			registration: cloneRegistration(instance.registration),
			client:       instance.pingClient,
		})
	}
	return targets
}

func (r *serviceRegistry) close() {
	var instances []*registeredInstance

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	for key, instance := range r.instances {
		delete(r.instances, key)
		instances = append(instances, instance)
	}
	r.mu.Unlock()

	for _, instance := range instances {
		if instance.connection != nil {
			_ = instance.connection.Close()
		}
	}
}

func (r *serviceRegistry) count(serviceType string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for key := range r.instances {
		if key.serviceType == serviceType {
			count++
		}
	}
	return count
}

func (r *serviceRegistry) instance(key instanceKey) (*gatewaypb.RegisterReq, time.Time, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	instance, ok := r.instances[key]
	if !ok {
		return nil, time.Time{}, false
	}
	return cloneRegistration(instance.registration), instance.lastPing, true
}

func cloneRegistration(registration *gatewaypb.RegisterReq) *gatewaypb.RegisterReq {
	return &gatewaypb.RegisterReq{
		Type:              registration.GetType(),
		Ip:                registration.GetIp(),
		Port:              registration.GetPort(),
		RegisterTime:      registration.GetRegisterTime(),
		Version:           registration.GetVersion(),
		Weight:            registration.GetWeight(),
		InstanceId:        registration.GetInstanceId(),
		RegistrationToken: registration.GetRegistrationToken(),
	}
}
