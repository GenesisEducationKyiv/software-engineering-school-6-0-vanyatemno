package controllers

import (
	"se-school/internal/config"
	"se-school/internal/controllers/middlewares"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all subscription-related routes on the given Gin engine
// under the /api base path, matching the swagger specification.
// CORS middleware is applied globally to allow cross-origin requests.
// Protected endpoints require a valid API key via the X-API-Key header.
func RegisterRoutes(r *gin.Engine, sc *SubscriptionController, cfg *config.Application) {
	r.Use(middlewares.CORSMiddleware())

	api := r.Group("/api", middlewares.APIKeyMiddleware(cfg.APIKey))
	{
		api.GET("/confirm/:token", sc.Confirm)
		api.GET("/unsubscribe/:token", sc.Unsubscribe)
		api.POST("/subscribe", sc.Subscribe)
		api.GET("/subscriptions", sc.GetSubscriptions)
	}
}
