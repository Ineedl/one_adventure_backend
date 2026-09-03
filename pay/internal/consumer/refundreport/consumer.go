package refundreport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"

	"github.com/go-sql-driver/mysql"
	"github.com/gogf/gf/v2/os/gtime"
	segmentkafka "github.com/segmentio/kafka-go"
	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"
	"pay/internal/dao"
)

const StatusRunning = "RUNNING"

type Consumer struct{ consumer *kafkakit.Consumer }

func New(ctx context.Context) (*Consumer, error) {
	config, err := kafkakit.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: kafkakit.NewConsumer(config, contractevent.RefundReportTopic)}, nil
}

func (c *Consumer) Run(ctx context.Context) error { return c.consumer.Run(ctx, c.handle) }
func (c *Consumer) Close() error                  { return c.consumer.Close() }

func (c *Consumer) handle(ctx context.Context, message segmentkafka.Message, commit kafkakit.CommitFunc) error {
	var event contractevent.RefundReport
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("decode refund report: %w", err)
	}
	if event.EventID == "" || event.RefundNo == "" || event.OrderID == 0 || event.PaymentOrderID == 0 || event.RefundAmount <= 0 {
		return fmt.Errorf("invalid refund report")
	}
	columns := dao.PaymentRefund.Columns()
	now := gtime.Now()
	_, err := dao.PaymentRefund.Ctx(ctx).Data(map[string]any{
		columns.Id: stableID(event.RefundNo), columns.RefundNo: event.RefundNo,
		columns.OrderId: event.OrderID, columns.PaymentOrderId: event.PaymentOrderID,
		columns.UserId: event.UserID, columns.RefundAmount: event.RefundAmount,
		columns.Reason: event.Reason, columns.Status: StatusRunning, columns.RetryCount: 0,
		columns.NextRetryAt: now, columns.CreatedAt: now, columns.UpdatedAt: now,
	}).Insert()
	if err != nil && !isDuplicate(err) {
		return fmt.Errorf("persist refund report: %w", err)
	}
	if err = commit(); err != nil {
		return fmt.Errorf("commit refund report: %w", err)
	}
	return nil
}

func isDuplicate(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func stableID(key string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(key))
	id := hasher.Sum64() & math.MaxInt64
	if id == 0 {
		return 1
	}
	return id
}
