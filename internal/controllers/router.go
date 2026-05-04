package controllers

import (
	"se-school/internal/config"
	"se-school/internal/controllers/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes registers all subscription-related routes on the given Gin engine
// under the /api base path, matching the swagger specification.
// CORS middleware is applied globally to allow cross-origin requests.
// Prometheus metrics middleware is applied globally to track HTTP request metrics.
// Protected endpoints require a valid API key via the X-API-Key header.
// Swagger UI is served at /swagger/*any.
// Prometheus metrics are served at /metrics.
func RegisterRoutes(r *gin.Engine, sc *SubscriptionController, cfg *config.Application) {
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.PrometheusMiddleware())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api", middlewares.APIKeyMiddleware(cfg.APIKey))
	{
		api.GET("/confirm/:token", sc.Confirm)
		api.GET("/unsubscribe/:token", sc.Unsubscribe)
		api.POST("/subscribe", sc.Subscribe)
		api.GET("/subscriptions", sc.GetSubscriptions)
	}
}
