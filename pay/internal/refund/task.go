package refund

import (
	"context"
	"errors"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"pay/internal/dao"
	"pay/internal/model/entity"
)

const (
	StatusRunning = "RUNNING"
	StatusSuccess = "SUCCESS"
	batchSize     = 1000
	retryDelay    = time.Minute
)

type QueryStatus string

const (
	QueryRunning QueryStatus = "RUNNING"
	QuerySuccess QueryStatus = "SUCCESS"
	QueryFailed  QueryStatus = "FAILED"
)

type Gateway interface {
	QueryRefund(context.Context, entity.PaymentRefund) (QueryStatus, error)
	Refund(context.Context, entity.PaymentRefund) error
}

// MockGateway is the placeholder for a future third-party refund adapter.
// Query leaves the task running; Refund intentionally always fails.
type MockGateway struct{}

func (MockGateway) QueryRefund(context.Context, entity.PaymentRefund) (QueryStatus, error) {
	return QueryRunning, nil
}
func (MockGateway) Refund(context.Context, entity.PaymentRefund) error {
	return errors.New("mock third-party refund failed")
}

type Task struct {
	gateway  Gateway
	interval time.Duration
}

func New(gateway Gateway) *Task { return &Task{gateway: gateway, interval: 5 * time.Second} }

func (t *Task) Run(ctx context.Context) error {
	if err := t.RunBatch(ctx); err != nil {
		g.Log().Errorf(ctx, "scan refund tasks: %v", err)
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := t.RunBatch(ctx); err != nil {
				g.Log().Errorf(ctx, "scan refund tasks: %v", err)
			}
		}
	}
}

func (t *Task) RunBatch(ctx context.Context) error {
	columns := dao.PaymentRefund.Columns()
	var tasks []entity.PaymentRefund
	if err := dao.PaymentRefund.Ctx(ctx).
		Where(columns.Status, StatusRunning).
		Where("next_retry_at IS NULL OR next_retry_at <= ?", gtime.Now()).
		OrderAsc(columns.Id).Limit(batchSize).Scan(&tasks); err != nil {
		return err
	}

	successIDs := make([]uint64, 0, len(tasks))
	toExecute := make([]entity.PaymentRefund, 0, len(tasks))
	// Finish every query decision before performing updates or refund calls.
	for _, task := range tasks {
		queryStatus, err := t.gateway.QueryRefund(ctx, task)
		if err == nil && queryStatus == QuerySuccess {
			successIDs = append(successIDs, task.Id)
			continue
		}
		g.Log().Warningf(ctx, "refund query not successful refund_no=%s status=%s error=%v", task.RefundNo, queryStatus, err)
		toExecute = append(toExecute, task)
	}

	if len(successIDs) > 0 {
		now := gtime.Now()
		if _, err := dao.PaymentRefund.Ctx(ctx).WhereIn(columns.Id, successIDs).Where(columns.Status, StatusRunning).
			Data(map[string]any{columns.Status: StatusSuccess, columns.RefundTime: now, columns.UpdatedAt: now, columns.LastError: ""}).Update(); err != nil {
			return err
		}
	}

	for _, task := range toExecute {
		err := t.gateway.Refund(ctx, task)
		if err == nil {
			continue
		}
		g.Log().Errorf(ctx, "execute refund failed refund_no=%s error=%v", task.RefundNo, err)
		now := gtime.Now()
		if _, updateErr := dao.PaymentRefund.Ctx(ctx).Where(columns.Id, task.Id).Where(columns.Status, StatusRunning).
			Data(map[string]any{columns.RetryCount: task.RetryCount + 1, columns.NextRetryAt: now.Add(retryDelay), columns.LastError: err.Error(), columns.UpdatedAt: now}).Update(); updateErr != nil {
			return updateErr
		}
	}
	return nil
}
