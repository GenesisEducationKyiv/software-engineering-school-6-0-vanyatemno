package middlewares

import (
	"net/http"
	"time"

	"se-school/internal/infrastructure/logging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// requestIDHeader is the header used to read/propagate the correlation ID.
	requestIDHeader = "X-Request-ID"
	// RequestIDContextKey is the Gin context key under which the request ID is stored.
	RequestIDContextKey = "request_id"
)

// RequestID ensures every request carries a correlation ID. An incoming
// X-Request-ID header is reused when present, otherwise a new UUID is generated.
// The ID is echoed on the response header, stored in the Gin context, and bound
// to a request-scoped logger that handlers can fetch via logging.FromContext.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Header(requestIDHeader, requestID)
		c.Set(RequestIDContextKey, requestID)

		reqLogger := zap.L().With(zap.String("request_id", requestID))
		ctx := logging.ContextWithLogger(c.Request.Context(), reqLogger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequestLogger emits one structured log entry per HTTP request using ECS-style
// field names. The level mirrors the response status: 5xx -> error,
// 4xx -> warn, otherwise info. It replaces Gin's default plain-text logger so
// that the entire log stream stays JSON and ingestible by the pipeline.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		fields := []zap.Field{
			zap.String("http.request.method", c.Request.Method),
			zap.String("url.path", path),
			zap.String("url.query", query),
			zap.Int("http.response.status_code", status),
			zap.Int("http.response.body.bytes", c.Writer.Size()),
			zap.Duration("event.duration", latency),
			zap.String("client.ip", c.ClientIP()),
			zap.String("user_agent.original", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error.message", c.Errors.String()))
		}

		logger := logging.FromContext(c.Request.Context())

		switch {
		case status >= http.StatusInternalServerError:
			logger.Error("http request", fields...)
		case status >= http.StatusBadRequest:
			logger.Warn("http request", fields...)
		default:
			logger.Info("http request", fields...)
		}
	}
}

// Recovery recovers from panics in downstream handlers, logs them as structured
// errors (with a stack trace) through the request-scoped logger, and responds
// with HTTP 500. It replaces gin.Recovery() to keep recovery logs structured.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logging.FromContext(c.Request.Context()).Error(
					"recovered from panic",
					zap.Any("panic", r),
					zap.Stack("stacktrace"),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()

		c.Next()
	}
}
