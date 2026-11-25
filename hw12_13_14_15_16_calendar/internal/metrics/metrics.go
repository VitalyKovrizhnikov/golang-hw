package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "calendar"

type Metrics struct {
	registry *prometheus.Registry

	httpRequests        *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpInFlight        prometheus.Gauge

	eventsCreated prometheus.Counter
	eventsUpdated prometheus.Counter
	eventsDeleted prometheus.Counter
	eventsListed  prometheus.Counter

	operationErrors *prometheus.CounterVec
}

func New() *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		registry: registry,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by method, route and status.",
		}, []string{"method", "route", "status"}),
		httpRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request processing duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
		httpInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Number of HTTP requests currently being processed.",
		}),
		eventsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "events",
			Name:      "created_total",
			Help:      "Total number of successfully created events.",
		}),
		eventsUpdated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "events",
			Name:      "updated_total",
			Help:      "Total number of successfully updated events.",
		}),
		eventsDeleted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "events",
			Name:      "deleted_total",
			Help:      "Total number of successfully deleted events.",
		}),
		eventsListed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "events",
			Name:      "listed_total",
			Help:      "Total number of events returned by list requests.",
		}),
		operationErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "event",
			Name:      "operation_errors_total",
			Help:      "Total number of failed event operations by operation.",
		}, []string{"operation"}),
	}

	registry.MustRegister(
		metrics.httpRequests,
		metrics.httpRequestDuration,
		metrics.httpInFlight,
		metrics.eventsCreated,
		metrics.eventsUpdated,
		metrics.eventsDeleted,
		metrics.eventsListed,
		metrics.operationErrors,
	)

	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) RequestStarted() {
	m.httpInFlight.Inc()
}

func (m *Metrics) RequestFinished(method, route string, status int, duration time.Duration) {
	statusCode := strconv.Itoa(status)
	m.httpRequests.WithLabelValues(method, route, statusCode).Inc()
	m.httpRequestDuration.WithLabelValues(method, route, statusCode).Observe(duration.Seconds())
	m.httpInFlight.Dec()
}

func (m *Metrics) EventCreated() {
	m.eventsCreated.Inc()
}

func (m *Metrics) EventUpdated() {
	m.eventsUpdated.Inc()
}

func (m *Metrics) EventDeleted() {
	m.eventsDeleted.Inc()
}

func (m *Metrics) EventsListed(count int) {
	m.eventsListed.Add(float64(count))
}

func (m *Metrics) EventOperationFailed(operation string) {
	m.operationErrors.WithLabelValues(operation).Inc()
}
