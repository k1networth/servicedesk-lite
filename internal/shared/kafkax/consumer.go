package kafkax

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	r *kafka.Reader
}

type ConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string

	// StartOffset controls where a NEW consumer group starts reading when it has no committed offsets.
	// Supported values: "first" | "last". Default: "last".
	StartOffset string

	MinBytes int
	MaxBytes int
}

func NewConsumer(cfg ConsumerConfig) *Consumer {
	minB := cfg.MinBytes
	maxB := cfg.MaxBytes
	if minB == 0 {
		minB = 1
	}
	if maxB == 0 {
		maxB = 10e6
	}

	start := kafka.LastOffset
	if strings.EqualFold(cfg.StartOffset, "first") {
		start = kafka.FirstOffset
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		StartOffset: start,
		MinBytes:    minB,
		MaxBytes:    maxB,
	})
	return &Consumer{r: r}
}

func (c *Consumer) Close() error { return c.r.Close() }

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.r.FetchMessage(ctx)
}

func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.r.CommitMessages(ctx, msgs...)
}
