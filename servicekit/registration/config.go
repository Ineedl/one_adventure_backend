package registration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	defaultRegisterTimeout      = 8 * time.Second
	defaultGatewayPingTimeout   = 30 * time.Second
	defaultRetryInitialInterval = time.Second
	defaultRetryMaxInterval     = 30 * time.Second
)

type ServiceInfo struct {
	Type       string
	IP         string
	Port       int
	Version    string
	Weight     int32
	InstanceID string
}

type Config struct {
	GatewayIP   string
	GatewayPort int
	Service     ServiceInfo

	RegisterTimeout      time.Duration
	GatewayPingTimeout   time.Duration
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.Service.InstanceID) == "" {
		c.Service.InstanceID = defaultInstanceID(c.Service.Type, c.Service.Port)
	}
	if c.RegisterTimeout <= 0 {
		c.RegisterTimeout = defaultRegisterTimeout
	}
	if c.GatewayPingTimeout <= 0 {
		c.GatewayPingTimeout = defaultGatewayPingTimeout
	}
	if c.RetryInitialInterval <= 0 {
		c.RetryInitialInterval = defaultRetryInitialInterval
	}
	if c.RetryMaxInterval <= 0 {
		c.RetryMaxInterval = defaultRetryMaxInterval
	}
	return c
}

func defaultInstanceID(serviceType string, port int) string {
	return sanitizeServiceName(serviceType) + "_" + machineCode(port)
}

func sanitizeServiceName(serviceType string) string {
	serviceType = strings.TrimSpace(strings.ToLower(serviceType))
	var builder strings.Builder
	lastUnderscore := false
	for _, character := range serviceType {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' {
			builder.WriteRune(character)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	name := strings.Trim(builder.String(), "_")
	if name == "" {
		return "service"
	}
	return name
}

func machineCode(port int) string {
	parts := make([]string, 0, 4)
	if override := strings.TrimSpace(os.Getenv("ONE_ADVENTURE_MACHINE_ID")); override != "" {
		parts = append(parts, "override="+override)
	} else {
		if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
			parts = append(parts, "hostname="+strings.TrimSpace(hostname))
		}
		if interfaces, err := net.Interfaces(); err == nil {
			hardwareAddresses := make([]string, 0, len(interfaces))
			for _, networkInterface := range interfaces {
				if networkInterface.Flags&net.FlagLoopback != 0 || len(networkInterface.HardwareAddr) == 0 {
					continue
				}
				hardwareAddresses = append(hardwareAddresses, networkInterface.HardwareAddr.String())
			}
			sort.Strings(hardwareAddresses)
			parts = append(parts, hardwareAddresses...)
		}
	}
	// The port distinguishes multiple instances of the same service running on
	// one machine. The raw host and network identifiers are never exposed.
	parts = append(parts, "port="+strconv.Itoa(port))
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:6])
}

func (c Config) validate() error {
	if strings.TrimSpace(c.GatewayIP) == "" {
		return fmt.Errorf("gateway ip is required")
	}
	if c.GatewayPort < 1 || c.GatewayPort > 65535 {
		return fmt.Errorf("gateway port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Service.Type) == "" {
		return fmt.Errorf("service type is required")
	}
	if net.ParseIP(strings.TrimSpace(c.Service.IP)) == nil {
		return fmt.Errorf("a valid service ip is required")
	}
	if c.Service.Port < 1 || c.Service.Port > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Service.InstanceID) == "" {
		return fmt.Errorf("service instance id is required")
	}
	if c.RetryMaxInterval < c.RetryInitialInterval {
		return fmt.Errorf("retry max interval must be greater than or equal to retry initial interval")
	}
	return nil
}
