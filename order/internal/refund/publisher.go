package refund

import (
	"context"
	"encoding/json"
	"fmt"

	contractevent "one_adventure_servicekit/api-contract/event"
	kafkakit "one_adventure_servicekit/kafka"
)

type Publisher struct{ producer *kafkakit.Producer }

func New(producer *kafkakit.Producer) *Publisher { return &Publisher{producer: producer} }

func (p *Publisher) Publish(ctx context.Context, event contractevent.RefundReport) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode refund report: %w", err)
	}
	if err = p.producer.Write(ctx, contractevent.RefundReportTopic, event.OrderNo, data); err != nil {
		return fmt.Errorf("publish refund report: %w", err)
	}
	return nil
}
