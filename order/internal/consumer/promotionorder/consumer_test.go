package promotionorder

import (
	"testing"
	"time"

	ordertimeout "order/internal/timeout"
)

func TestOrderAmount(t *testing.T) {
	amount, err := orderAmount(250, 3)
	if err != nil {
		t.Fatalf("calculate order amount: %v", err)
	}
	if amount != 750 {
		t.Fatalf("unexpected amount: %d", amount)
	}
}

func TestEventExpiresAt(t *testing.T) {
	createdAt := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	if got := eventExpiresAt(createdAt.UnixMilli()); !got.Equal(createdAt.Add(ordertimeout.OrderTTL)) {
		t.Fatalf("unexpected expiration: %v", got)
	}
}
