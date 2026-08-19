package order

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	orderpb "one_adventure_rpc/proto/order"
	"order/internal/compensation"
	"order/internal/dao"
	"order/internal/model/entity"
	"order/internal/orderstate"
	ordertimeout "order/internal/timeout"
)

type Service struct {
	orderpb.UnimplementedOrderServiceServer
	publisher *compensation.Publisher
}

func NewService(publisher *compensation.Publisher) *Service { return &Service{publisher: publisher} }
func (s *Service) Pay(ctx context.Context, req *orderpb.PayReq) (*orderpb.OrderResp, error) {
	return s.updateStatus(ctx, req.GetOrderNo(), req.GetRequestId(), orderstate.Paid, "")
}
func (s *Service) Cancel(ctx context.Context, req *orderpb.CancelReq) (*orderpb.OrderResp, error) {
	return s.updateStatus(ctx, req.GetOrderNo(), req.GetRequestId(), orderstate.Canceled, "canceled")
}

func (s *Service) updateStatus(ctx context.Context, orderNo, requestID, target, reason string) (*orderpb.OrderResp, error) {
	columns := dao.Orders.Columns()
	query := dao.Orders.Ctx(ctx)
	if orderNo != "" {
		query = query.Where(columns.OrderNo, orderNo)
	} else if requestID != "" {
		query = query.Where(columns.RequestId, requestID)
	} else {
		return nil, status.Error(codes.InvalidArgument, "order_no or request_id is required")
	}
	var order entity.Orders
	if err := query.Scan(&order); err != nil {
		return nil, status.Errorf(codes.Internal, "query order: %v", err)
	}
	if order.OrderId == 0 {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	if order.Status != orderstate.PendingPay {
		return nil, status.Errorf(codes.FailedPrecondition, "order status is %s", order.Status)
	}
	result, err := dao.Orders.Ctx(ctx).Where(columns.OrderId, order.OrderId).Where(columns.Status, orderstate.PendingPay).Data(columns.Status, target).Update()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update order status: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return nil, status.Error(codes.Aborted, "order status changed concurrently")
	}
	if _, err = g.Redis().ZRem(ctx, ordertimeout.Key, order.OrderNo); err != nil {
		return nil, status.Errorf(codes.Internal, "remove order timeout: %v", err)
	}
	if reason != "" {
		if err = s.publisher.Publish(ctx, order, reason); err != nil {
			return nil, status.Errorf(codes.Unavailable, "publish compensation: %v", err)
		}
	}
	return &orderpb.OrderResp{Success: true, OrderNo: order.OrderNo, Status: target}, nil
}
