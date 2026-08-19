package promotionorder

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	segmentkafka "github.com/segmentio/kafka-go"
	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"

	"order/internal/dao"
	"order/internal/orderstate"
	ordertimeout "order/internal/timeout"
)

const promotionOrderType = 1

type Consumer struct {
	consumer *kafkakit.Consumer
}

func New(ctx context.Context) (*Consumer, error) {
	config, err := kafkakit.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: kafkakit.NewConsumer(config, contractevent.PromotionOrderCreateTopic)}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx, c.handle)
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}

func (c *Consumer) handle(ctx context.Context, message segmentkafka.Message, commit kafkakit.CommitFunc) error {
	var event contractevent.PromotionOrderCreate
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("decode promotion order event: %w", err)
	}
	if err := validateEvent(event); err != nil {
		return err
	}

	columns := dao.Orders.Columns()
	count, err := dao.Orders.Ctx(ctx).Where(columns.RequestId, event.RequestID).Count()
	if err != nil {
		return fmt.Errorf("check promotion order idempotency: %w", err)
	}
	if count == 0 {
		amount, err := orderAmount(event.Price, event.PayNum)
		if err != nil {
			return err
		}
		orderNo := "ORD-" + uuid.NewString()
		_, err = dao.Orders.Ctx(ctx).Data(map[string]any{
			columns.OrderNo:      orderNo,
			columns.UserId:       event.UserID,
			columns.ProductId:    event.ProductID,
			columns.Quantity:     event.PayNum,
			columns.Amount:       amount,
			columns.CurrencyType: event.CurrencyType,
			columns.OrderType:    promotionOrderType,
			columns.Status:       orderstate.PendingPay,
			columns.PromotionId:  event.PromotionID,
			columns.RequestId:    event.RequestID,
		}).Insert()
		if err != nil {
			return fmt.Errorf("create promotion order: %w", err)
		}
		if err = ordertimeout.Add(ctx, orderNo, eventExpiresAt(event.CreatedAt)); err != nil {
			return err
		}
	} else {
		var orderNo string
		if err = dao.Orders.Ctx(ctx).Fields(columns.OrderNo).Where(columns.RequestId, event.RequestID).Scan(&orderNo); err != nil {
			return fmt.Errorf("query existing promotion order id: %w", err)
		}
		if err = ordertimeout.Add(ctx, orderNo, eventExpiresAt(event.CreatedAt)); err != nil {
			return err
		}
	}
	if err = commit(); err != nil {
		return fmt.Errorf("commit promotion order event: %w", err)
	}
	return nil
}

func eventExpiresAt(createdAt int64) time.Time {
	return time.UnixMilli(createdAt).Add(ordertimeout.OrderTTL)
}

func validateEvent(event contractevent.PromotionOrderCreate) error {
	if strings.TrimSpace(event.RequestID) == "" || event.UserID == 0 || event.ProductID == 0 || event.PromotionID == 0 || event.PayNum <= 0 || event.Price < 0 || event.CreatedAt <= 0 {
		return fmt.Errorf("invalid promotion order event")
	}
	return nil
}

func orderAmount(price int64, quantity int32) (int64, error) {
	if price > 0 && int64(quantity) > math.MaxInt64/price {
		return 0, fmt.Errorf("promotion order amount overflows int64")
	}
	return price * int64(quantity), nil
}
