package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ── HTTP ──────────────────────────────────────────────────────────────────
	// Общий счётчик запросов. Лейбл path использует шаблон маршрута (/v1/applicants/:id),
	// а не реальный URL — иначе каждый ID абитуриента создавал бы отдельную временную серию.
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// Гистограмма задержек — позволяет в Grafana считать p50/p95/p99 через histogram_quantile().
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"method", "path"},
	)

	// ── Документы ─────────────────────────────────────────────────────────────
	// Сколько документов загружено, по категориям (passport, diploma и т.д.).
	// Помогает понять нагрузку на MinIO и какие типы документов чаще всего загружают.
	DocumentsUploadedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "manage_documents_uploaded_total",
			Help: "Total documents uploaded, by category",
		},
		[]string{"category"},
	)

	// Результат AI-обработки документа после прохождения через RabbitMQ → extraction-service.
	// "success" — данные извлечены, "error" — модель вернула ошибку или timeout.
	DocumentsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "manage_documents_processed_total",
			Help: "Total documents processed by AI pipeline",
		},
		[]string{"status"}, // "success" | "error"
	)

	// ── SSE ───────────────────────────────────────────────────────────────────
	// Текущее количество активных SSE-соединений (Gauge, потому что может уменьшаться).
	// Резкий рост → клиенты не получают статус и переподключаются.
	SSEActiveClients = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "manage_sse_active_clients",
			Help: "Number of currently active SSE connections",
		},
	)

	// ── RabbitMQ ──────────────────────────────────────────────────────────────
	// Сколько задач отправлено в очередь на обработку.
	RabbitmqPublishedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "manage_rabbitmq_published_total",
			Help: "Total messages published to RabbitMQ",
		},
	)

	// Сколько задач извлечено из очереди и обработано.
	// Если consumed сильно отстаёт от published — очередь растёт, extraction-service не справляется.
	RabbitmqConsumedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "manage_rabbitmq_consumed_total",
			Help: "Total messages consumed from RabbitMQ",
		},
		[]string{"status"}, // "success" | "error"
	)
)
