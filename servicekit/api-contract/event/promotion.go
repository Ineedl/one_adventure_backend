// Package event defines Kafka event contracts shared by producers and consumers.
package event

const (
	PromotionOrderCreateTopic     = "promotion_order_create"
	PromotionOrderCompensateTopic = "promotion_order_compensate"
)

// PromotionOrderCreate is published after promotion stock is deducted.
type PromotionOrderCreate struct {
	RequestID    string `json:"request_id"`
	TraceID      string `json:"trace_id"`
	PromotionID  uint64 `json:"promotion_id"`
	ProductID    uint64 `json:"product_id"`
	UserID       uint64 `json:"user_id"`
	PayNum       int32  `json:"pay_num"`
	Price        int64  `json:"price"`
	CurrencyType int32  `json:"currency_type"`
	CreatedAt    int64  `json:"created_at"`
}

// PromotionOrderCompensate restores promotion stock for an unpaid terminal order.
type PromotionOrderCompensate struct {
	RequestID   string `json:"request_id"`
	TraceID     string `json:"trace_id"`
	OrderNo     string `json:"order_no"`
	PromotionID uint64 `json:"promotion_id"`
	ProductID   uint64 `json:"product_id"`
	UserID      uint64 `json:"user_id"`
	PayNum      int32  `json:"pay_num"`
	Reason      string `json:"reason"`
	CreatedAt   int64  `json:"created_at"`
}
