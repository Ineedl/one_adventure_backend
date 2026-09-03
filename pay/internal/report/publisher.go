package report

import (
	"context"
	"encoding/json"
	"fmt"

	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"
)

type Publisher struct{ producer *kafkakit.Producer }

func New(producer *kafkakit.Producer) *Publisher { return &Publisher{producer: producer} }

func (p *Publisher) PublishPay(ctx context.Context, event contractevent.PayReport) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode pay report: %w", err)
	}
	if err = p.producer.Write(ctx, contractevent.PayReportTopic, event.OrderNo, data); err != nil {
		return fmt.Errorf("publish pay report: %w", err)
	}
	return nil
}
