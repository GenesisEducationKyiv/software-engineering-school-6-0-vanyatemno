package middlewares

import (
	"strconv"
	"time"

	"se-school/internal/metrics"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware returns a Gin middleware that records HTTP request metrics
// (total requests, request duration, and in-flight requests) using Prometheus.
// The path label is normalized to the route pattern (e.g. /api/confirm/:token)
// to avoid high-cardinality label values.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		method := c.Request.Method

		duration := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
