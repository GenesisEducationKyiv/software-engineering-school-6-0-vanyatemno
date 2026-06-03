package middlewares

import (
	"errors"
	"net/http"
	"se-school/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandlerMiddleware runs after handlers and translates any error
// pushed via c.Error(err) into the appropriate HTTP response. Handlers
// should call c.Error(err) and return without writing a response when
// they want this middleware to map the error.
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		switch {
		case errors.Is(err, models.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found on GitHub"})
		case errors.Is(err, models.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		case errors.Is(err, models.ErrAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "Email already subscribed to this repository"})
		default:
			zap.L().Error("unhandled service error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
	}
}
