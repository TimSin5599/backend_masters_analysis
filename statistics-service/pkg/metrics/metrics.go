package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"method", "path"},
	)

	// Время выполнения тяжёлых DB-запросов по типу.
	// overview и dynamics делают разные агрегации — важно видеть каждый отдельно.
	StatsQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stats_query_duration_seconds",
			Help:    "Duration of statistics DB queries",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"query"}, // "overview" | "dynamics"
	)
)
