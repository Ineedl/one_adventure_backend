package registration

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	randv2 "math/rand/v2"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	gatewaypb "one_adventure_rpc/proto/gateway"
	serviceping "one_adventure_servicekit/ping"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrAlreadyRunning = errors.New("registration manager is already running")

type Option func(*Manager)

func WithLogger(logger *slog.Logger) Option {
	return func(manager *Manager) {
		if logger != nil {
			manager.logger = logger
		}
	}
}

func WithDialOptions(options ...grpc.DialOption) Option {
	return func(manager *Manager) {
		if len(options) > 0 {
			manager.dialOptions = append([]grpc.DialOption(nil), options...)
		}
	}
}

type registrationAttempt struct {
	token string
}

type Manager struct {
	config      Config
	logger      *slog.Logger
	dialOptions []grpc.DialOption
	pingService *serviceping.Service

	stateMu sync.RWMutex
	state   State

	attemptMu sync.RWMutex
	attempt   *registrationAttempt

	runMu   sync.Mutex
	running bool
}

func New(config Config, options ...Option) (*Manager, error) {
	config = config.withDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	pingService, err := serviceping.New(config.Service.Type, config.Service.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("create ping service: %w", err)
	}
	manager := &Manager{
		config:      config,
		logger:      slog.Default(),
		dialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
		pingService: pingService,
		state:       StateStarting,
	}
	for _, option := range options {
		option(manager)
	}
	return manager, nil
}

func (m *Manager) RegisterPingService(server grpc.ServiceRegistrar) {
	m.pingService.Register(server)
}

func (m *Manager) State() State {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.state
}

func (m *Manager) Run(ctx context.Context) error {
	if !m.beginRun() {
		return ErrAlreadyRunning
	}
	defer func() {
		m.setState(StateStopped)
		m.endRun()
	}()

	connection, err := grpc.NewClient(
		net.JoinHostPort(strings.TrimSpace(m.config.GatewayIP), strconv.Itoa(m.config.GatewayPort)),
		m.dialOptions...,
	)
	if err != nil {
		return fmt.Errorf("create gateway grpc client: %w", err)
	}
	defer connection.Close()
	client := gatewaypb.NewGatewayServiceClient(connection)

	retryInterval := m.config.RetryInitialInterval
	for ctx.Err() == nil {
		m.setState(StateRegistering)
		activated, attemptErr := m.register(ctx, client)
		if activated {
			retryInterval = m.config.RetryInitialInterval
			m.setState(StateActive)
			m.logger.Info("service registration active",
				"service_type", m.config.Service.Type,
				"instance_id", m.config.Service.InstanceID,
				"gateway_ping_timeout", m.config.GatewayPingTimeout,
			)
			attemptErr = m.monitorGatewayPings(ctx)
		}
		m.clearAttempt()
		if ctx.Err() != nil {
			break
		}
		m.logger.Error("service registration inactive, scheduling retry",
			"service_type", m.config.Service.Type,
			"instance_id", m.config.Service.InstanceID,
			"retry_interval", retryInterval,
			"error", attemptErr,
		)
		m.setState(StateRetryWaiting)
		if !waitContext(ctx, jitter(retryInterval)) {
			break
		}
		retryInterval = minDuration(retryInterval*2, m.config.RetryMaxInterval)
	}

	return ctx.Err()
}

func (m *Manager) register(ctx context.Context, client gatewaypb.GatewayServiceClient) (bool, error) {
	token, err := newToken()
	if err != nil {
		return false, fmt.Errorf("generate registration token: %w", err)
	}
	attempt := &registrationAttempt{token: token}
	m.setAttempt(attempt)

	registerCtx, cancel := context.WithTimeout(ctx, m.config.RegisterTimeout)
	response, err := client.RegisterGateway(registerCtx, &gatewaypb.RegisterReq{
		Type:              strings.TrimSpace(m.config.Service.Type),
		Ip:                strings.TrimSpace(m.config.Service.IP),
		Port:              int32(m.config.Service.Port),
		RegisterTime:      time.Now().UnixMilli(),
		Version:           m.config.Service.Version,
		Weight:            m.config.Service.Weight,
		InstanceId:        strings.TrimSpace(m.config.Service.InstanceID),
		RegistrationToken: token,
	})
	cancel()
	if err != nil {
		return false, fmt.Errorf("register gateway: %w", err)
	}
	if response.GetCode() != 0 {
		return false, fmt.Errorf("register gateway rejected with code %d: %s", response.GetCode(), response.GetMessage())
	}
	return true, nil
}

// monitorGatewayPings runs only after registration succeeds. Registration and
// retry states therefore never trigger the missing-Ping reconnect logic.
func (m *Manager) monitorGatewayPings(ctx context.Context) error {
	timer := time.NewTimer(m.config.GatewayPingTimeout)
	defer timer.Stop()
	for {
		select {
		case <-m.pingService.Activity():
			timer.Reset(m.config.GatewayPingTimeout)
		case <-timer.C:
			return fmt.Errorf("gateway ping not received within %s", m.config.GatewayPingTimeout)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *Manager) setAttempt(attempt *registrationAttempt) {
	m.attemptMu.Lock()
	m.attempt = attempt
	m.attemptMu.Unlock()
	m.pingService.Activate(attempt.token)
}

func (m *Manager) clearAttempt() {
	m.attemptMu.Lock()
	attempt := m.attempt
	m.attempt = nil
	m.attemptMu.Unlock()
	if attempt != nil {
		m.pingService.Deactivate(attempt.token)
	}
}

func (m *Manager) setState(state State) {
	m.stateMu.Lock()
	m.state = state
	m.stateMu.Unlock()
}

func (m *Manager) beginRun() bool {
	m.runMu.Lock()
	defer m.runMu.Unlock()
	if m.running {
		return false
	}
	m.running = true
	return true
}

func (m *Manager) endRun() {
	m.clearAttempt()
	m.runMu.Lock()
	m.running = false
	m.runMu.Unlock()
}

func newToken() (string, error) {
	var value [16]byte
	if _, err := cryptorand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func waitContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func jitter(duration time.Duration) time.Duration {
	if duration <= 1 {
		return duration
	}
	spread := duration / 5
	return duration - spread + time.Duration(randv2.Int64N(int64(spread*2)+1))
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
