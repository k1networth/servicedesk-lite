package kafkax

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	w *kafka.Writer
}

type ProducerConfig struct {
	Brokers      []string
	Topic        string
	ClientID     string
	WriteTimeout time.Duration
}

func NewProducer(cfg ProducerConfig) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		BatchTimeout: 50 * time.Millisecond,
	}
	return &Producer{w: w}
}

func (p *Producer) Close() error { return p.w.Close() }

func (p *Producer) Produce(ctx context.Context, key []byte, value []byte, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return p.w.WriteMessages(ctx, kafka.Message{Key: key, Value: value})
}
