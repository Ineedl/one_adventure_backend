package discovery

import (
	"context"
	"testing"
)

func TestRegistrarPutIfAbsentValidatesInputBeforeEtcdRequest(t *testing.T) {
	registrar := &Registrar{}
	if _, err := registrar.PutIfAbsent(context.Background(), " ", "value", 10); err == nil {
		t.Fatal("PutIfAbsent() error = nil, want empty key error")
	}
	if _, err := registrar.PutIfAbsent(context.Background(), "/lock", "value", 0); err == nil {
		t.Fatal("PutIfAbsent() error = nil, want invalid ttl error")
	}
}
