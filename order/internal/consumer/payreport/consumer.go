package payreport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	segmentkafka "github.com/segmentio/kafka-go"
	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"
	"order/internal/compensation"
	"order/internal/dao"
	"order/internal/model/entity"
	"order/internal/orderstate"
	"order/internal/refund"
	ordertimeout "order/internal/timeout"
)

type Consumer struct {
	consumer     *kafkakit.Consumer
	refund       *refund.Publisher
	compensation *compensation.Publisher
}

func New(ctx context.Context, refundPublisher *refund.Publisher, compensationPublisher *compensation.Publisher) (*Consumer, error) {
	config, err := kafkakit.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: kafkakit.NewConsumer(config, contractevent.PayReportTopic), refund: refundPublisher, compensation: compensationPublisher}, nil
}

func (c *Consumer) Run(ctx context.Context) error { return c.consumer.Run(ctx, c.handle) }
func (c *Consumer) Close() error                  { return c.consumer.Close() }

func (c *Consumer) handle(ctx context.Context, message segmentkafka.Message, commit kafkakit.CommitFunc) error {
	var event contractevent.PayReport
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("decode pay report: %w", err)
	}
	if event.EventID == "" || event.OrderNo == "" || event.TransactionNo == "" {
		return fmt.Errorf("invalid pay report")
	}
	columns := dao.Orders.Columns()
	var order entity.Orders
	if err := dao.Orders.Ctx(ctx).Where(columns.OrderNo, event.OrderNo).Scan(&order); err != nil {
		return fmt.Errorf("query pay report order: %w", err)
	}
	if order.OrderId == 0 {
		return fmt.Errorf("pay report order %s not found", event.OrderNo)
	}

	switch strings.ToUpper(event.Status) {
	case orderstate.Paid:
		if order.Status == orderstate.Paid {
			break // duplicate report
		}
		if order.Status != orderstate.PendingPay {
			refundEvent := contractevent.RefundReport{
				EventID: event.EventID + ":refund", RefundNo: "R" + event.TransactionNo,
				OrderID: order.OrderId, OrderNo: order.OrderNo, PaymentOrderID: event.PaymentOrderID,
				PaymentNo: event.PaymentNo, UserID: order.UserId, RefundAmount: event.Amount,
				Reason: "payment succeeded after order status became " + order.Status, CreatedAt: time.Now().UnixMilli(),
			}
			if err := c.refund.Publish(ctx, refundEvent); err != nil {
				return err
			}
			break
		}
		result, err := dao.Orders.Ctx(ctx).Where(columns.OrderId, order.OrderId).Where(columns.Status, orderstate.PendingPay).Data(columns.Status, orderstate.Paid).Update()
		if err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return fmt.Errorf("mark order paid concurrently")
		}
	case orderstate.Canceled:
		if order.Status == orderstate.Canceled {
			// The previous attempt may have updated the order and then failed before
			// publishing compensation/committing Kafka. The downstream compensation
			// operation is idempotent, so publish it again on redelivery.
			if c.compensation != nil {
				if err := c.compensation.Publish(ctx, order, event.Reason); err != nil {
					return err
				}
			}
			break
		}
		if order.Status != orderstate.PendingPay {
			break
		}
		result, err := dao.Orders.Ctx(ctx).Where(columns.OrderId, order.OrderId).Where(columns.Status, orderstate.PendingPay).Data(columns.Status, orderstate.Canceled).Update()
		if err != nil {
			return fmt.Errorf("cancel order from pay report: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return fmt.Errorf("cancel order concurrently")
		}
		if c.compensation != nil {
			order.Status = orderstate.Canceled
			if err = c.compensation.Publish(ctx, order, event.Reason); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported pay report status %q", event.Status)
	}
	if _, err := g.Redis().ZRem(ctx, ordertimeout.Key, order.OrderNo); err != nil {
		return fmt.Errorf("remove order timeout: %w", err)
	}
	if err := commit(); err != nil {
		return fmt.Errorf("commit pay report: %w", err)
	}
	return nil
}
