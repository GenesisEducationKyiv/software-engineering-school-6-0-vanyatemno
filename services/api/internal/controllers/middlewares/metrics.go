package middlewares

import (
	"strconv"
	"time"

	"se-school/internal/metrics"

	"github.com/gin-gonic/gin"
)

// The path label is normalized to the route pattern (e.g. /api/confirm/:token)
// to avoid high-cardinality label values.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip the scrape endpoint itself: instrumenting it would make the
		// in-flight gauge always read 1 (the scrape counts itself) and pollute
		// the request total/duration with scrape traffic.
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

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
		metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	}
}
