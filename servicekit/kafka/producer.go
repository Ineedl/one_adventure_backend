package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct{ writer *kafka.Writer }

func NewProducer(c Config) *Producer {
	c = c.WithDefaults()
	return &Producer{writer: &kafka.Writer{Addr: kafka.TCP(c.Brokers...), Balancer: &kafka.LeastBytes{}, WriteTimeout: c.WriteTimeout}}
}

func (p *Producer) Write(ctx context.Context, topic, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(key), Value: value, Time: time.Now()})
}
func (p *Producer) Close() error { return p.writer.Close() }
