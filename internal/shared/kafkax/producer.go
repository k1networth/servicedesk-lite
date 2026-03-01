package kafkax

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	mu        sync.Mutex
	w         *kafka.Writer
	cfg       ProducerConfig
	lastReset time.Time
}

type ProducerConfig struct {
	Brokers      []string
	Topic        string
	ClientID     string
	WriteTimeout time.Duration
}

func NewProducer(cfg ProducerConfig) *Producer {
	p := &Producer{cfg: cfg}
	p.w = newWriter(cfg)
	return p
}

func newWriter(cfg ProducerConfig) *kafka.Writer {
	// kafka-go caches broker metadata; when broker addresses change (e.g. after fixing
	// advertised.listeners), a long metadata TTL may keep clients stuck until restart.
	// We keep TTL low to self-heal without manual restarts.
	tr := &kafka.Transport{
		ClientID:    cfg.ClientID,
		MetadataTTL: 10 * time.Second,
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		BatchTimeout: 50 * time.Millisecond,
		Transport:    tr,
	}
}

func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.w == nil {
		return nil
	}
	err := p.w.Close()
	p.w = nil
	return err
}

func (p *Producer) Produce(ctx context.Context, key []byte, value []byte, timeout time.Duration) error {
	if timeout <= 0 {
		if p.cfg.WriteTimeout > 0 {
			timeout = p.cfg.WriteTimeout
		} else {
			timeout = 5 * time.Second
		}
	}

	write := func() error {
		p.mu.Lock()
		w := p.w
		p.mu.Unlock()
		if w == nil {
			return context.Canceled
		}
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return w.WriteMessages(cctx, kafka.Message{Key: key, Value: value})
	}

	if err := write(); err != nil {
		// Self-heal common failure mode: Kafka advertised.listeners changed while
		// this producer is running (stale metadata). Recreate writer and retry once.
		if shouldReset(err) {
			p.resetOnce()
			return write()
		}
		return err
	}
	return nil
}

func shouldReset(err error) bool {
	if err == nil {
		return false
	}
	// Heuristic: reset on typical network/metadata problems.
	s := strings.ToLower(err.Error())
	suspects := []string{
		"dial tcp",
		"connection refused",
		"i/o timeout",
		"eof",
		"broken pipe",
		"transport is closing",
		"not leader",
		"unknown broker",
		"failed to dial",
	}
	for _, sub := range suspects {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func (p *Producer) resetOnce() {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Rate-limit resets to avoid tight loops.
	if time.Since(p.lastReset) < 2*time.Second {
		return
	}
	if p.w != nil {
		_ = p.w.Close()
	}
	p.w = newWriter(p.cfg)
	p.lastReset = time.Now()
}
