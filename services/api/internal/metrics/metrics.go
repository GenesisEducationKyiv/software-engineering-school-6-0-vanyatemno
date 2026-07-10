package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "se_school",
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "se_school",
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	CronJobRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "cron_job_runs_total",
			Help:      "Total number of cron job executions",
		},
		[]string{"job", "status"},
	)

	// Buckets span seconds to minutes since background jobs run far longer than HTTP requests.
	CronJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "se_school",
			Name:      "cron_job_duration_seconds",
			Help:      "Duration of cron job executions in seconds",
			Buckets:   []float64{0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
		[]string{"job"},
	)

	RepoCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "repo_check_total",
			Help:      "Total number of individual repository version checks",
		},
		[]string{"status"},
	)
)
