package kafkax

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	mu  sync.Mutex
	r   *kafka.Reader
	cfg ConsumerConfig
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
	c := &Consumer{cfg: cfg}
	c.r = newReader(cfg)
	return c
}

func newReader(cfg ConsumerConfig) *kafka.Reader {
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

	// NOTE: we set MaxWait/Backoffs so FetchMessage doesn't hang forever on transient broker/metadata issues.
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		StartOffset:    start,
		MinBytes:       minB,
		MaxBytes:       maxB,
		MaxWait:        500 * time.Millisecond,
		ReadBackoffMin: 100 * time.Millisecond,
		ReadBackoffMax: 1 * time.Second,
	})
	return r
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.r == nil {
		return nil
	}
	err := c.r.Close()
	c.r = nil
	return err
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	c.mu.Lock()
	r := c.r
	c.mu.Unlock()
	return r.FetchMessage(ctx)
}

func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	c.mu.Lock()
	r := c.r
	c.mu.Unlock()
	return r.CommitMessages(ctx, msgs...)
}

// Reopen closes the underlying reader and recreates it using the original config.
// Useful when broker metadata becomes stale (e.g., after advertised.listeners changes) or on transient network errors.
func (c *Consumer) Reopen() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.r != nil {
		_ = c.r.Close()
	}
	c.r = newReader(c.cfg)
}
