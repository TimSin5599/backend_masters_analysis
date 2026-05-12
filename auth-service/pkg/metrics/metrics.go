package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP метрики — собираются автоматически в LoggingMiddleware

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

	// Auth-специфичные метрики

	LoginAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_login_attempts_total",
			Help: "Total login attempts",
		},
		[]string{"result"}, // "success" | "invalid_credentials" | "internal_error"
	)

	TokenRefreshTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_token_refresh_total",
			Help: "Total token refresh attempts",
		},
		[]string{"result"}, // "success" | "invalid_token" | "internal_error"
	)

	LogoutTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_logout_total",
			Help: "Total logout requests",
		},
	)

	PasswordChangesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_password_changes_total",
			Help: "Total password change attempts",
		},
		[]string{"result"}, // "success" | "invalid_credentials" | "not_found" | "internal_error"
	)
)
