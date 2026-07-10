package controllers

import (
	"se-school/internal/config"
	"se-school/internal/controllers/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes wires the subscription routes under /api (API-key protected),
// plus the global middleware stack and the /swagger and /metrics endpoints.
func RegisterRoutes(r *gin.Engine, sc *SubscriptionController, cfg *config.Application) {
	r.Use(middlewares.RequestID())
	r.Use(middlewares.RequestLogger())
	r.Use(middlewares.Recovery())
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.PrometheusMiddleware())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api", middlewares.ErrorHandlerMiddleware(), middlewares.APIKeyMiddleware(cfg.APIKey))
	{
		api.GET("/confirm/:token", sc.Confirm)
		api.GET("/unsubscribe/:token", sc.Unsubscribe)
		api.POST("/subscribe", sc.Subscribe)
		api.GET("/subscribe/status/:sagaId", sc.Status)
		api.GET("/subscriptions", sc.GetSubscriptions)
	}
}
