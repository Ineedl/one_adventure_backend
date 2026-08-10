package config

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	defaultJWTPrivateKeyPath     = "key/private.pem"
	defaultJWTPublicKeyPath      = "key/public.pem"
	defaultJWTIssuer             = "one-adventure-user"
	defaultAccessTokenExpire     = 24 * time.Hour
	defaultRefreshTokenExpire    = 30 * 24 * time.Hour
	defaultRefreshTokenKeyPrefix = "user:refresh_token:"
)

// JWTConfig is the user service's global JWT configuration.
type JWTConfig struct {
	PrivateKeyPath        string        `json:"privateKeyPath"`
	PublicKeyPath         string        `json:"publicKeyPath"`
	Issuer                string        `json:"issuer"`
	AccessTokenExpire     time.Duration `json:"accessTokenExpire"`
	RefreshTokenExpire    time.Duration `json:"refreshTokenExpire"`
	RefreshTokenKeyPrefix string        `json:"refreshTokenKeyPrefix"`
}

func LoadJWT(ctx context.Context) (JWTConfig, error) {
	value, err := g.Cfg().Get(ctx, "jwt")
	if err != nil {
		return JWTConfig{}, fmt.Errorf("read jwt config: %w", err)
	}
	if value.IsEmpty() {
		return JWTConfig{}, fmt.Errorf("jwt config is required")
	}
	var cfg JWTConfig
	if err = value.Scan(&cfg); err != nil {
		return JWTConfig{}, fmt.Errorf("parse jwt config: %w", err)
	}
	cfg = cfg.WithDefaults()
	if err = cfg.Validate(); err != nil {
		return JWTConfig{}, fmt.Errorf("validate jwt config: %w", err)
	}
	return cfg, nil
}

func (c JWTConfig) Validate() error {
	c = c.WithDefaults()
	if c.PrivateKeyPath == "" {
		return fmt.Errorf("private key path is required")
	}
	if c.PublicKeyPath == "" {
		return fmt.Errorf("public key path is required")
	}
	if c.AccessTokenExpire <= 0 {
		return fmt.Errorf("access token expire must be greater than zero")
	}
	if c.RefreshTokenExpire <= 0 {
		return fmt.Errorf("refresh token expire must be greater than zero")
	}
	return nil
}

func (c JWTConfig) WithDefaults() JWTConfig {
	if c.PrivateKeyPath == "" {
		c.PrivateKeyPath = defaultJWTPrivateKeyPath
	}
	if c.PublicKeyPath == "" {
		c.PublicKeyPath = defaultJWTPublicKeyPath
	}
	if c.Issuer == "" {
		c.Issuer = defaultJWTIssuer
	}
	if c.AccessTokenExpire <= 0 {
		c.AccessTokenExpire = defaultAccessTokenExpire
	}
	if c.RefreshTokenExpire <= 0 {
		c.RefreshTokenExpire = defaultRefreshTokenExpire
	}
	if c.RefreshTokenKeyPrefix == "" {
		c.RefreshTokenKeyPrefix = defaultRefreshTokenKeyPrefix
	}
	return c
}
