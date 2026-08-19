package compensation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	obslog "one_adventure_observability_log"
	tracekit "one_adventure_observability_trace/trace"
	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"

	"order/internal/model/entity"
)

type Publisher struct{ producer *kafkakit.Producer }

func New(producer *kafkakit.Producer) *Publisher { return &Publisher{producer: producer} }

func (p *Publisher) Publish(ctx context.Context, order entity.Orders, reason string) error {
	event := contractevent.PromotionOrderCompensate{
		RequestID: order.RequestId, TraceID: tracekit.TraceID(ctx), OrderNo: order.OrderNo,
		PromotionID: order.PromotionId, ProductID: order.ProductId,
		UserID: order.UserId, PayNum: int32(order.Quantity), Reason: reason, CreatedAt: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode promotion order compensation: %w", err)
	}
	if err = p.producer.Write(ctx, contractevent.PromotionOrderCompensateTopic, order.OrderNo, data); err != nil {
		return fmt.Errorf("publish promotion order compensation: %w", err)
	}
	obslog.Info(ctx, "published promotion order compensation", map[string]any{
		"order_no": order.OrderNo, "request_id": order.RequestId, "promotion_id": order.PromotionId,
		"product_id": order.ProductId, "user_id": order.UserId, "pay_num": order.Quantity, "reason": reason,
	})
	return nil
}
