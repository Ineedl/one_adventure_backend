package event

const (
	PayReportTopic    = "pay_report"
	RefundReportTopic = "refund_report"
)

// PayReport asks order to apply the result of a payment operation.
type PayReport struct {
	EventID        string `json:"event_id"`
	OrderID        uint64 `json:"order_id"`
	OrderNo        string `json:"order_no"`
	PaymentOrderID int64  `json:"payment_order_id"`
	PaymentNo      string `json:"payment_no"`
	TransactionNo  string `json:"transaction_no"`
	UserID         uint64 `json:"user_id"`
	Amount         int64  `json:"amount"` // cents
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	CreatedAt      int64  `json:"created_at"`
}

// RefundReport asks pay to persist and execute a refund task.
type RefundReport struct {
	EventID        string `json:"event_id"`
	RefundNo       string `json:"refund_no"`
	OrderID        uint64 `json:"order_id"`
	OrderNo        string `json:"order_no"`
	PaymentOrderID int64  `json:"payment_order_id"`
	PaymentNo      string `json:"payment_no"`
	UserID         uint64 `json:"user_id"`
	RefundAmount   int64  `json:"refund_amount"` // cents
	Reason         string `json:"reason"`
	CreatedAt      int64  `json:"created_at"`
}
