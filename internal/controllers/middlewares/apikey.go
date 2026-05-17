package middlewares

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

const apiKeyHeader = "X-API-Key"

// APIKeyMiddleware returns a Gin middleware that validates requests against
// the configured API key. The key is expected in the X-API-Key header.
// If the configured apiKey is empty, the middleware is a no-op (all requests pass).
func APIKeyMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}

		provided := c.GetHeader(apiKeyHeader)
		if provided == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing API key",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Invalid API key",
			})
			return
		}

		c.Next()
	}
}
