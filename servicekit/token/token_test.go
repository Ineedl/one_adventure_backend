package token

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func TestRS256GenerateVerifyAndParse(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, privateKey)})
	publicData, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKIXPublicKey() error = %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicData})

	generator, err := NewRS256Generator(privatePEM, "test-issuer", time.Hour)
	if err != nil {
		t.Fatalf("NewRS256Generator() error = %v", err)
	}
	verifier, err := NewRS256Verifier(publicPEM, "test-issuer")
	if err != nil {
		t.Fatalf("NewRS256Verifier() error = %v", err)
	}
	fixedNow := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	generator.now = func() time.Time { return fixedNow }
	verifier.now = func() time.Time { return fixedNow }

	value, err := generator.Generate(UserInfo{ID: 7, Username: "alice", Status: 0})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	claims, err := verifier.VerifyAndParse(value)
	if err != nil {
		t.Fatalf("VerifyAndParse() error = %v", err)
	}
	if claims.UserInfo.ID != 7 || claims.UserInfo.Username != "alice" || claims.Subject != "7" {
		t.Fatalf("Claims = %+v", claims)
	}

	value = value[:len(value)-1] + "x"
	if _, err = verifier.VerifyAndParse(value); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("VerifyAndParse(tampered) error = %v", err)
	}
}

func TestUserInfoGRPCMetadata(t *testing.T) {
	want := UserInfo{ID: 7, Username: "alice", Status: 0}
	ctx, err := WithOutgoingUserInfo(WithUserInfo(context.Background(), want))
	if err != nil {
		t.Fatalf("WithOutgoingUserInfo() error = %v", err)
	}
	outgoing, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("outgoing metadata is missing")
	}
	incoming := metadata.NewIncomingContext(context.Background(), outgoing)
	got, ok := UserInfoFromIncomingContext(incoming)
	if !ok || got != want {
		t.Fatalf("UserInfoFromIncomingContext() = %+v, %v; want %+v, true", got, ok, want)
	}
}

func TestRS256VerifierRejectsExpiredToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)})
	generator, _ := NewRS256Generator(privatePEM, "issuer", time.Second)
	verifier, _ := NewRS256Verifier(publicPEM, "issuer")
	base := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	generator.now = func() time.Time { return base }
	value, _ := generator.Generate(UserInfo{ID: 1, Username: "alice"})
	verifier.now = func() time.Time { return base.Add(2 * time.Second) }
	if _, err := verifier.VerifyAndParse(value); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("VerifyAndParse(expired) error = %v", err)
	}
}

func mustPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	data, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}
	return data
}
