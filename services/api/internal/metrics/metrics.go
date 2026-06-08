package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts the total number of HTTP requests, partitioned by method, path, and status code.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration observes the duration of HTTP requests in seconds, partitioned by method and path.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "se_school",
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// HTTPRequestsInFlight tracks the number of HTTP requests currently being processed.
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "se_school",
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)
)
