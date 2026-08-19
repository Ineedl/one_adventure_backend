package ordercompensate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gogf/gf/v2/frame/g"
	segmentkafka "github.com/segmentio/kafka-go"
	obslog "one_adventure_observability_log"
	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"
	promotionscript "promotion/script"
)

type Consumer struct{ consumer *kafkakit.Consumer }

func New(ctx context.Context) (*Consumer, error) {
	config, err := kafkakit.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: kafkakit.NewConsumer(config, contractevent.PromotionOrderCompensateTopic)}, nil
}

func (c *Consumer) Run(ctx context.Context) error { return c.consumer.Run(ctx, c.handle) }
func (c *Consumer) Close() error                  { return c.consumer.Close() }

func (c *Consumer) handle(ctx context.Context, message segmentkafka.Message, commit kafkakit.CommitFunc) error {
	var event contractevent.PromotionOrderCompensate
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("decode promotion compensation event: %w", err)
	}
	if event.RequestID == "" || event.PromotionID == 0 || event.ProductID == 0 || event.UserID == 0 || event.PayNum <= 0 {
		return fmt.Errorf("invalid promotion compensation event")
	}
	stockKey := fmt.Sprintf("promotion:%d:%d", event.PromotionID, event.ProductID)
	limitKey := fmt.Sprintf("promotion_limit:%d:%d:%d", event.PromotionID, event.ProductID, event.UserID)
	idempotencyKey := "promotion:compensate:" + event.RequestID
	result, err := g.Redis().GroupScript().Eval(ctx, promotionscript.SeckillCompensate, 3, []string{stockKey, limitKey, idempotencyKey}, []any{event.PayNum})
	if err != nil {
		return fmt.Errorf("execute promotion compensation: %w", err)
	}
	if result.Int() < 0 && result.Int() != -3 {
		return fmt.Errorf("promotion compensation failed with code %d", result.Int())
	}
	if err = commit(); err != nil {
		return fmt.Errorf("commit promotion compensation event: %w", err)
	}
	obslog.Info(ctx, "promotion order compensated", map[string]any{
		"order_no": event.OrderNo, "request_id": event.RequestID, "promotion_id": event.PromotionID,
		"product_id": event.ProductID, "user_id": event.UserID, "pay_num": event.PayNum,
		"reason": event.Reason, "duplicate": result.Int() == -3,
	})
	return nil
}
