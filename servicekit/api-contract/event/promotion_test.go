package event

import (
	"encoding/json"
	"testing"
)

func TestPromotionOrderCreateJSONContract(t *testing.T) {
	event := PromotionOrderCreate{
		RequestID:    "request-1",
		TraceID:      "trace-1",
		PromotionID:  1,
		ProductID:    2,
		UserID:       3,
		PayNum:       4,
		Price:        500,
		CurrencyType: 1,
		CreatedAt:    123456789,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal promotion order create event: %v", err)
	}
	const expected = `{"request_id":"request-1","trace_id":"trace-1","promotion_id":1,"product_id":2,"user_id":3,"pay_num":4,"price":500,"currency_type":1,"created_at":123456789}`
	if string(data) != expected {
		t.Fatalf("unexpected event JSON: %s", data)
	}
}

func TestPromotionOrderCompensateTopic(t *testing.T) {
	if PromotionOrderCompensateTopic != "promotion_order_compensate" {
		t.Fatalf("unexpected compensation topic: %s", PromotionOrderCompensateTopic)
	}
}
