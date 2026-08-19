package timeout

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
	obslog "one_adventure_observability_log"

	"order/internal/compensation"
	"order/internal/dao"
	"order/internal/model/entity"
	"order/internal/orderstate"
)

const (
	Key          = "order:timeout"
	OrderTTL     = 15 * time.Minute
	scanInterval = time.Second
	scanBatch    = 100
)

type Task struct{ publisher *compensation.Publisher }

func New(publisher *compensation.Publisher) *Task { return &Task{publisher: publisher} }

func Add(ctx context.Context, orderNo string, expiresAt time.Time) error {
	_, err := g.Redis().ZAdd(ctx, Key, nil, gredis.ZAddMember{
		Score:  float64(expiresAt.UnixMilli()),
		Member: orderNo,
	})
	if err != nil {
		return fmt.Errorf("add order timeout: %w", err)
	}
	return nil
}

func (t *Task) Run(ctx context.Context) error {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if err := t.scan(ctx, now); err != nil {
				obslog.Error(ctx, "scan expired orders failed", map[string]any{"error": err.Error()})
			}
		}
	}
}

func (t *Task) scan(ctx context.Context, now time.Time) error {
	offset, count := 0, scanBatch
	members, err := g.Redis().ZRange(ctx, Key, 0, now.UnixMilli(), gredis.ZRangeOption{
		ByScore: true,
		Limit:   &gredis.ZRangeOptionLimit{Offset: &offset, Count: &count},
	})
	if err != nil {
		return fmt.Errorf("read expired orders from redis: %w", err)
	}
	for _, member := range members {
		columns := dao.Orders.Columns()
		var order entity.Orders
		if err = dao.Orders.Ctx(ctx).Where(columns.OrderNo, member.String()).Scan(&order); err != nil {
			return fmt.Errorf("query expired order %s: %w", member.String(), err)
		}
		if order.OrderId == 0 {
			if _, err = g.Redis().ZRem(ctx, Key, member.String()); err != nil {
				return fmt.Errorf("remove missing order %s from redis: %w", member.String(), err)
			}
			continue
		}
		updateResult, updateErr := dao.Orders.Ctx(ctx).
			Where(columns.OrderNo, member.String()).
			Where(columns.Status, orderstate.PendingPay).
			Data(columns.Status, orderstate.Expired).
			Update()
		if updateErr != nil {
			return fmt.Errorf("expire order %s: %w", member.String(), updateErr)
		}
		updated, updateErr := updateResult.RowsAffected()
		if updateErr != nil {
			return fmt.Errorf("read expired order %s update result: %w", member.String(), updateErr)
		}
		shouldCompensate := updated == 1 || order.Status == orderstate.Expired || order.Status == orderstate.Canceled
		if shouldCompensate {
			reason := "expired"
			if order.Status == orderstate.Canceled {
				reason = "canceled"
			}
			if err = t.publisher.Publish(ctx, order, reason); err != nil {
				return err
			}
		}
		if _, err = g.Redis().ZRem(ctx, Key, member.String()); err != nil {
			return fmt.Errorf("remove expired order %s from redis: %w", member.String(), err)
		}
	}
	return nil
}
