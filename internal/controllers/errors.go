package controllers

import (
	"errors"
	"net/http"
	"se-school/internal/infrastructure/db"
	"se-school/internal/repositories"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v84/github"
	"go.uber.org/zap"
)

// handleServiceError maps known service/repository errors to appropriate HTTP responses.
func handleServiceError(c *gin.Context, err error) {
	if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response.StatusCode == http.StatusNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found on GitHub"})
		return
	}
	if errors.Is(err, repositories.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}
	if db.IsDuplicateKeyError(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already subscribed to this repository"})
		return
	}

	zap.L().Error("unhandled service error", zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
