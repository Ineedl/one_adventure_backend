package config

import (
	"testing"
	"time"

	"github.com/gogf/gf/v2/container/gvar"
)

func TestJWTConfigDefaults(t *testing.T) {
	cfg := (JWTConfig{}).WithDefaults()
	if cfg.PrivateKeyPath != defaultJWTPrivateKeyPath ||
		cfg.PublicKeyPath != defaultJWTPublicKeyPath ||
		cfg.Issuer != defaultJWTIssuer ||
		cfg.AccessTokenExpire != defaultAccessTokenExpire ||
		cfg.RefreshTokenExpire != defaultRefreshTokenExpire ||
		cfg.RefreshTokenKeyPrefix != defaultRefreshTokenKeyPrefix {
		t.Fatalf("JWT defaults = %+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("JWTConfig.Validate() error = %v", err)
	}
}

func TestJWTConfigScan(t *testing.T) {
	var cfg JWTConfig
	err := gvar.New(map[string]any{
		"privateKeyPath":        "key/private.pem",
		"publicKeyPath":         "key/public.pem",
		"issuer":                "test",
		"accessTokenExpire":     "2h",
		"refreshTokenExpire":    "720h",
		"refreshTokenKeyPrefix": "test:refresh_token:",
	}).Scan(&cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if cfg.AccessTokenExpire != 2*time.Hour || cfg.RefreshTokenExpire != 30*24*time.Hour {
		t.Fatalf("JWT config = %+v", cfg)
	}
}
