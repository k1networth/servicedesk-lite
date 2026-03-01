package httpx

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ctxKeyRoute struct{}

type routeHolder struct {
	route string
}

func WithRoute(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := r.Context().Value(ctxKeyRoute{}).(*routeHolder); ok && h != nil {
			h.route = route
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyRoute{}, &routeHolder{route: route})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type metricsRecorder struct {
	http.ResponseWriter
	status int
}

func (w *metricsRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

type Metrics struct {
	reqTotal    *prometheus.CounterVec
	reqLatency  *prometheus.HistogramVec
	req5xxTotal prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		reqTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"route", "method", "status"},
		),
		req5xxTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "http_requests_5xx_total",
				Help: "Total number of HTTP 5xx responses.",
			},
		),
		reqLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"route", "method"},
		),
	}

	reg.MustRegister(m.reqTotal, m.reqLatency, m.req5xxTotal)
	return m
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		holder := &routeHolder{route: r.URL.Path}
		ctx := context.WithValue(r.Context(), ctxKeyRoute{}, holder)
		r = r.WithContext(ctx)

		start := time.Now()
		mw := &metricsRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(mw, r)

		route := holder.route
		status := strconv.Itoa(mw.status)
		dur := time.Since(start).Seconds()

		m.reqTotal.WithLabelValues(route, r.Method, status).Inc()
		m.reqLatency.WithLabelValues(route, r.Method).Observe(dur)
		if mw.status >= 500 {
			m.req5xxTotal.Inc()
		}
	})
}
