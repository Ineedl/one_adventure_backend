// Package token provides shared access-token generation and verification.
package token

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

const UserInfoMetadataKey = "x-one-adventure-user-info"

type userInfoContextKey struct{}

// UserInfo is the common user identity carried by access tokens and refresh
// token records. New shared identity fields can be added here later.
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Status   int    `json:"status"`
}

func WithUserInfo(ctx context.Context, userInfo UserInfo) context.Context {
	return context.WithValue(ctx, userInfoContextKey{}, userInfo)
}

func UserInfoFromContext(ctx context.Context) (UserInfo, bool) {
	userInfo, ok := ctx.Value(userInfoContextKey{}).(UserInfo)
	return userInfo, ok
}

// WithOutgoingUserInfo copies authenticated identity into outgoing gRPC metadata.
func WithOutgoingUserInfo(ctx context.Context) (context.Context, error) {
	userInfo, ok := UserInfoFromContext(ctx)
	if !ok {
		return ctx, nil
	}
	value, err := json.Marshal(userInfo)
	if err != nil {
		return nil, fmt.Errorf("marshal user info metadata: %w", err)
	}
	return metadata.AppendToOutgoingContext(ctx, UserInfoMetadataKey, string(value)), nil
}

func UserInfoFromIncomingContext(ctx context.Context) (UserInfo, bool) {
	values := metadata.ValueFromIncomingContext(ctx, UserInfoMetadataKey)
	if len(values) == 0 {
		return UserInfo{}, false
	}
	var userInfo UserInfo
	if err := json.Unmarshal([]byte(values[0]), &userInfo); err != nil || userInfo.ID <= 0 {
		return UserInfo{}, false
	}
	return userInfo, true
}

type Claims struct {
	Subject   string   `json:"sub"`
	UserInfo  UserInfo `json:"user_info"`
	Issuer    string   `json:"iss"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

type Generator interface {
	Generate(userInfo UserInfo) (string, error)
}

type Verifier interface {
	VerifyAndParse(value string) (*Claims, error)
}

type RS256Generator struct {
	privateKey *rsa.PrivateKey
	issuer     string
	expire     time.Duration
	now        func() time.Time
}

type RS256Verifier struct {
	publicKey *rsa.PublicKey
	issuer    string
	now       func() time.Time
}

func NewRS256GeneratorFromFile(privateKeyPath, issuer string, expire time.Duration) (*RS256Generator, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read RS256 private key: %w", err)
	}
	return NewRS256Generator(keyData, issuer, expire)
}

func NewRS256Generator(privateKeyPEM []byte, issuer string, expire time.Duration) (*RS256Generator, error) {
	if expire <= 0 {
		return nil, fmt.Errorf("token expiration must be greater than zero")
	}
	privateKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	return &RS256Generator{privateKey: privateKey, issuer: issuer, expire: expire, now: time.Now}, nil
}

func NewRS256VerifierFromFile(publicKeyPath, issuer string) (*RS256Verifier, error) {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read RS256 public key: %w", err)
	}
	return NewRS256Verifier(keyData, issuer)
}

func NewRS256Verifier(publicKeyPEM []byte, issuer string) (*RS256Verifier, error) {
	publicKey, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}
	return &RS256Verifier{publicKey: publicKey, issuer: issuer, now: time.Now}, nil
}

func (g *RS256Generator) Generate(userInfo UserInfo) (string, error) {
	if userInfo.ID <= 0 {
		return "", fmt.Errorf("user id must be greater than zero")
	}
	now := g.now()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(Claims{
		Subject: strconv.FormatInt(userInfo.ID, 10), UserInfo: userInfo, Issuer: g.issuer,
		IssuedAt: now.Unix(), ExpiresAt: now.Add(g.expire).Unix(),
	})
	if err != nil {
		return "", err
	}
	unsigned := encodeSegment(header) + "." + encodeSegment(payload)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, g.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign RS256 token: %w", err)
	}
	return unsigned + "." + encodeSegment(signature), nil
}

func (v *RS256Verifier) VerifyAndParse(value string) (*Claims, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	headerData, err := decodeSegment(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err = json.Unmarshal(headerData, &header); err != nil || header.Algorithm != "RS256" || header.Type != "JWT" {
		return nil, ErrInvalidToken
	}
	signature, err := decodeSegment(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err = rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return nil, ErrInvalidToken
	}
	payload, err := decodeSegment(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err = json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, ErrInvalidToken
	}
	if claims.Subject == "" || claims.UserInfo.ID <= 0 || claims.Subject != strconv.FormatInt(claims.UserInfo.ID, 10) ||
		claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt <= v.now().Unix() {
		return nil, ErrExpiredToken
	}
	return &claims, nil
}

func parseRSAPrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("decode RS256 private key: invalid PEM data")
	}
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		privateKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("RS256 private key is not RSA")
		}
		return privateKey, nil
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse RS256 private key: %w", err)
	}
	return privateKey, nil
}

func parseRSAPublicKey(keyData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("decode RS256 public key: invalid PEM data")
	}
	if parsed, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		publicKey, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("RS256 public key is not RSA")
		}
		return publicKey, nil
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse RS256 public key: %w", err)
	}
	return publicKey, nil
}

func encodeSegment(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }

func decodeSegment(value string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(value) }
