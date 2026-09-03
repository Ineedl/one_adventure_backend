package pay

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	paypb "one_adventure_rpc/proto/pay"
	contractevent "one_adventure_servicekit/api-contract/event"
	"pay/internal/dao"
	"pay/internal/model/entity"
	"pay/internal/paymentstate"
	"pay/internal/report"
)

// Service owns payment data. OrderService is only notified after the local
// payment transaction commits, so it never observes an uncommitted payment.
type Service struct {
	paypb.UnimplementedPayServiceServer
	reporter *report.Publisher
}

func NewService(reporter *report.Publisher) *Service { return &Service{reporter: reporter} }

func stableNo(prefix, key string) string {
	return prefix + strings.ReplaceAll(uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String(), "-", "")
}

func stableID(key string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(key))
	id := int64(hasher.Sum64() & math.MaxInt64)
	if id == 0 {
		return 1
	}
	return id
}

func decimalAmount(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }
func amountCents(amount float64) int64 { return int64(math.Round(amount * 100)) }

func (s *Service) CreateTransaction(ctx context.Context, req *paypb.CreateTransactionReq) (*paypb.TransactionResp, error) {
	if req.GetRequestId() == "" || req.GetOrderNo() == "" || req.GetAmount() <= 0 || req.GetChannel() == "" || req.GetCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id, order_no, positive amount, currency and channel are required")
	}
	transactionNo := stableNo("T", req.GetRequestId())
	var transaction entity.PaymentTransaction
	err := dao.PaymentTransaction.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if err := tx.Model("payment_transaction").Ctx(ctx).Where("transaction_no", transactionNo).Scan(&transaction); err != nil {
			return err
		}
		if transaction.TransactionNo != "" {
			if transaction.OrderNo != req.GetOrderNo() || amountCents(transaction.RequestAmount) != req.GetAmount() || transaction.Channel != req.GetChannel() {
				return status.Error(codes.AlreadyExists, "request_id was used with different transaction parameters")
			}
			return nil
		}

		var paymentOrder entity.PaymentOrder
		if err := tx.Model("payment_order").Ctx(ctx).Where("order_no", req.GetOrderNo()).LockUpdate().Scan(&paymentOrder); err != nil {
			return err
		}
		if paymentOrder.PaymentNo == "" {
			paymentOrder = entity.PaymentOrder{PaymentNo: stableNo("P", req.GetOrderNo()), OrderNo: req.GetOrderNo(), UserId: int64(req.GetUserId()), Amount: float64(req.GetAmount()) / 100, Currency: req.GetCurrency(), Status: paymentstate.OrderPending}
			now := gtime.Now()
			orderData := gdb.Map{"id": stableID(paymentOrder.PaymentNo), "payment_no": paymentOrder.PaymentNo, "order_no": paymentOrder.OrderNo, "user_id": paymentOrder.UserId, "amount": decimalAmount(req.GetAmount()), "currency": paymentOrder.Currency, "status": paymentOrder.Status, "created_at": now, "updated_at": now}
			if _, err := tx.Model("payment_order").Ctx(ctx).Data(orderData).Insert(); err != nil {
				return err
			}
		} else if amountCents(paymentOrder.Amount) != req.GetAmount() || paymentOrder.UserId != int64(req.GetUserId()) || paymentOrder.Currency != req.GetCurrency() {
			return status.Error(codes.FailedPrecondition, "payment parameters do not match the existing payment order")
		} else if paymentOrder.Status != paymentstate.OrderPending {
			return status.Errorf(codes.FailedPrecondition, "payment order status is %s", paymentOrder.Status)
		}

		transaction = entity.PaymentTransaction{TransactionNo: transactionNo, PaymentNo: paymentOrder.PaymentNo, OrderNo: req.GetOrderNo(), Channel: req.GetChannel(), RequestAmount: float64(req.GetAmount()) / 100, Status: paymentstate.TransactionPending}
		now := gtime.Now()
		transactionData := gdb.Map{"id": stableID(transaction.TransactionNo), "transaction_no": transaction.TransactionNo, "payment_no": transaction.PaymentNo, "order_no": transaction.OrderNo, "channel": transaction.Channel, "request_amount": decimalAmount(req.GetAmount()), "status": transaction.Status, "created_at": now, "updated_at": now}
		_, err := tx.Model("payment_transaction").Ctx(ctx).Data(transactionData).Insert()
		return err
	})
	if err != nil {
		return nil, rpcError("create transaction", err)
	}
	return transactionResp(&transaction), nil
}

func (s *Service) CancelTransaction(ctx context.Context, req *paypb.CancelTransactionReq) (*paypb.TransactionResp, error) {
	if req.GetTransactionNo() == "" || req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "transaction_no and request_id are required")
	}
	var transaction entity.PaymentTransaction
	var paymentOrder entity.PaymentOrder
	err := dao.PaymentTransaction.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if err := tx.Model("payment_transaction").Ctx(ctx).Where("transaction_no", req.GetTransactionNo()).LockUpdate().Scan(&transaction); err != nil {
			return err
		}
		if transaction.TransactionNo == "" {
			return status.Error(codes.NotFound, "transaction not found")
		}
		if transaction.Status == paymentstate.TransactionCanceled {
			return nil
		}
		if transaction.Status != paymentstate.TransactionPending {
			return status.Errorf(codes.FailedPrecondition, "transaction status is %s", paymentstate.TransactionName(transaction.Status))
		}
		if _, err := tx.Model("payment_transaction").Ctx(ctx).Where("transaction_no", transaction.TransactionNo).Data(gdb.Map{"status": paymentstate.TransactionCanceled, "response_content": req.GetReason(), "updated_at": gtime.Now()}).Update(); err != nil {
			return err
		}
		if _, err := tx.Model("payment_order").Ctx(ctx).Where("payment_no", transaction.PaymentNo).Where("status", paymentstate.OrderPending).Data(gdb.Map{"status": paymentstate.OrderCanceled, "updated_at": gtime.Now()}).Update(); err != nil {
			return err
		}
		transaction.Status = paymentstate.TransactionCanceled
		if err := tx.Model("payment_order").Ctx(ctx).Where("payment_no", transaction.PaymentNo).Scan(&paymentOrder); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, rpcError("cancel transaction", err)
	}
	if paymentOrder.PaymentNo == "" {
		if err := dao.PaymentOrder.Ctx(ctx).Where("payment_no", transaction.PaymentNo).Scan(&paymentOrder); err != nil {
			return nil, rpcError("query canceled payment order", err)
		}
	}
	if err := s.publishPayReport(ctx, transaction, paymentOrder, "CANCELLED", req.GetReason()); err != nil {
		return nil, err
	}
	return transactionResp(&transaction), nil
}

func (s *Service) ThirdPartyCallback(ctx context.Context, req *paypb.ThirdPartyCallbackReq) (*paypb.TransactionResp, error) {
	target := strings.ToUpper(req.GetStatus())
	if req.GetTransactionNo() == "" || (target != paymentstate.OrderSuccess && target != paymentstate.OrderFailed) {
		return nil, status.Error(codes.InvalidArgument, "transaction_no and status SUCCESS or FAILED are required")
	}
	var transaction entity.PaymentTransaction
	var paymentOrder entity.PaymentOrder
	err := dao.PaymentTransaction.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if err := tx.Model("payment_transaction").Ctx(ctx).Where("transaction_no", req.GetTransactionNo()).LockUpdate().Scan(&transaction); err != nil {
			return err
		}
		if transaction.TransactionNo == "" {
			return status.Error(codes.NotFound, "transaction not found")
		}
		targetCode := paymentstate.TransactionFailed
		if target == paymentstate.OrderSuccess {
			targetCode = paymentstate.TransactionSuccess
		}
		if transaction.Status == targetCode {
			return nil
		}
		if transaction.Status != paymentstate.TransactionPending {
			return status.Errorf(codes.FailedPrecondition, "transaction status is %s", paymentstate.TransactionName(transaction.Status))
		}
		now := gtime.Now()
		data := gdb.Map{"status": targetCode, "callback_time": now, "callback_content": req.GetRawData(), "response_content": req.GetThirdPartyTransactionNo(), "updated_at": now}
		if targetCode == paymentstate.TransactionSuccess {
			data["paid_amount"] = decimalAmount(amountCents(transaction.RequestAmount))
		}
		if _, err := tx.Model("payment_transaction").Ctx(ctx).Where("transaction_no", transaction.TransactionNo).Data(data).Update(); err != nil {
			return err
		}
		orderData := gdb.Map{"status": target, "updated_at": now}
		if targetCode == paymentstate.TransactionSuccess {
			orderData["paid_time"] = now
		}
		if _, err := tx.Model("payment_order").Ctx(ctx).Where("payment_no", transaction.PaymentNo).Where("status", paymentstate.OrderPending).Data(orderData).Update(); err != nil {
			return err
		}
		transaction.Status = targetCode
		if err := tx.Model("payment_order").Ctx(ctx).Where("payment_no", transaction.PaymentNo).Scan(&paymentOrder); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, rpcError("process callback", err)
	}
	if target == paymentstate.OrderSuccess {
		if paymentOrder.PaymentNo == "" {
			if err := dao.PaymentOrder.Ctx(ctx).Where("payment_no", transaction.PaymentNo).Scan(&paymentOrder); err != nil {
				return nil, rpcError("query callback payment order", err)
			}
		}
		if err := s.publishPayReport(ctx, transaction, paymentOrder, "PAID", ""); err != nil {
			return nil, err
		}
	}
	return transactionResp(&transaction), nil
}

func (s *Service) publishPayReport(ctx context.Context, transaction entity.PaymentTransaction, paymentOrder entity.PaymentOrder, target, reason string) error {
	if s.reporter == nil {
		return status.Error(codes.Unavailable, "pay report publisher is not configured")
	}
	event := contractevent.PayReport{
		EventID: transaction.TransactionNo + ":" + target, OrderNo: transaction.OrderNo,
		PaymentOrderID: paymentOrder.Id, PaymentNo: transaction.PaymentNo,
		TransactionNo: transaction.TransactionNo, UserID: uint64(paymentOrder.UserId),
		Amount: amountCents(transaction.RequestAmount), Status: target, Reason: reason,
		CreatedAt: gtime.Now().UnixMilli(),
	}
	if err := s.reporter.PublishPay(ctx, event); err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	return nil
}

func transactionResp(t *entity.PaymentTransaction) *paypb.TransactionResp {
	return &paypb.TransactionResp{Success: true, PaymentNo: t.PaymentNo, TransactionNo: t.TransactionNo, Status: paymentstate.TransactionName(t.Status)}
}

func rpcError(operation string, err error) error {
	if _, ok := status.FromError(err); ok && status.Code(err) != codes.Unknown {
		return err
	}
	return status.Error(codes.Internal, fmt.Sprintf("%s: %v", operation, err))
}
