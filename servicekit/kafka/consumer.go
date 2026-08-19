package kafka

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
)

// CommitFunc commits the current message offset. Calling it repeatedly only
// performs one Kafka commit and returns the first commit result.
type CommitFunc func() error

// MessageHandler decides whether and when to commit by calling commit.
type MessageHandler func(context.Context, kafka.Message, CommitFunc) error
type Consumer struct{ reader *kafka.Reader }

func NewConsumer(c Config, topic string) *Consumer {
	c = c.WithDefaults()
	return &Consumer{reader: kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.Brokers,
		Dialer:   &kafka.Dialer{Timeout: c.ReadTimeout},
		GroupID:  c.Group,
		Topic:    topic,
		MinBytes: c.MinBytes,
		MaxBytes: c.MaxBytes,
		MaxWait:  c.MaxWait,
	})}
}

func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}
		var once sync.Once
		var commitErr error
		commit := func() error {
			once.Do(func() {
				commitErr = c.reader.CommitMessages(ctx, msg)
			})
			return commitErr
		}
		if err = handler(ctx, msg, commit); err != nil {
			return err
		}
	}
}
func (c *Consumer) Close() error { return c.reader.Close() }
