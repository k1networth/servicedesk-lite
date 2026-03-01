package outbox

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	PublishedTotal *prometheus.CounterVec
	FailedTotal    *prometheus.CounterVec
	DeadTotal      *prometheus.CounterVec
	LagSeconds     prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		PublishedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "outbox_published_total", Help: "Published outbox events."},
			[]string{"event_type"},
		),
		FailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "outbox_failed_total", Help: "Failed outbox publish attempts."},
			[]string{"event_type"},
		),
		DeadTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "outbox_dead_total", Help: "Outbox events moved to dead/failed state."},
			[]string{"event_type"},
		),
		LagSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "outbox_lag_seconds", Help: "Lag in seconds for oldest pending outbox event."},
		),
	}
	reg.MustRegister(m.PublishedTotal, m.FailedTotal, m.DeadTotal, m.LagSeconds)

	// Ensure time series exist even when counters are still zero.
	// Without this, Grafana panels may show "No data" for rate/total queries
	// until the first failure/dead event happens.
	for _, et := range []string{"ticket.created"} {
		m.PublishedTotal.WithLabelValues(et).Add(0)
		m.FailedTotal.WithLabelValues(et).Add(0)
		m.DeadTotal.WithLabelValues(et).Add(0)
	}
	return m
}
