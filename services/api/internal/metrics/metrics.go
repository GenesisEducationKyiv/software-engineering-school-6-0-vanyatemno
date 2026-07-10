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

	// HTTPRequestDuration observes the duration of HTTP requests in seconds, partitioned by method, path, and status code.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "se_school",
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestsInFlight tracks the number of HTTP requests currently being processed.
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "se_school",
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	// CronJobRunsTotal counts cron job executions, partitioned by job name and outcome (success/error).
	CronJobRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "cron_job_runs_total",
			Help:      "Total number of cron job executions",
		},
		[]string{"job", "status"},
	)

	// CronJobDuration observes the duration of cron job executions in seconds, partitioned by job name.
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

	// RepoCheckTotal counts individual repository checks within a cron run, partitioned by outcome (success/error).
	RepoCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "repo_check_total",
			Help:      "Total number of individual repository version checks",
		},
		[]string{"status"},
	)

	// RepoNotifyTotal counts per-subscription repository-update notifications,
	// partitioned by outcome. "success": delivered and last_seen_tag advanced.
	// "error": the notifier failed to deliver — last_seen_tag is left unchanged so
	// the subscriber is retried on the next cron run. "tag_error": delivery
	// succeeded but advancing last_seen_tag failed.
	RepoNotifyTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "se_school",
			Name:      "repo_notify_total",
			Help:      "Total number of per-subscription repository-update notifications",
		},
		[]string{"status"},
	)
)
