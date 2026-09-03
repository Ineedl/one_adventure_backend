package paymentstate

const (
	OrderPending  = "WAIT_PAY"
	OrderSuccess  = "SUCCESS"
	OrderFailed   = "FAILED"
	OrderCanceled = "CLOSED"

	TransactionPending    = 0 // CREATED
	TransactionProcessing = 1
	TransactionSuccess    = 2
	TransactionFailed     = 3
	TransactionCanceled   = 4 // CLOSED
)

func TransactionName(status int) string {
	switch status {
	case TransactionSuccess:
		return OrderSuccess
	case TransactionFailed:
		return OrderFailed
	case TransactionCanceled:
		return OrderCanceled
	case TransactionProcessing:
		return "PROCESSING"
	default:
		return OrderPending
	}
}
