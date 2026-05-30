package main

import (
	"context"
	"log"
	"se-school/internal/config"
	"se-school/internal/controllers"
	cronScheduler "se-school/internal/cron"
	"se-school/internal/infrastructure/db"
	"se-school/internal/infrastructure/logging"
	redisInfra "se-school/internal/infrastructure/redis"
	"se-school/internal/integrations/github"
	"se-school/internal/notifications"
	"se-school/internal/notifications/mailer"
	"se-school/internal/notifications/templates"
	codeRepo "se-school/internal/repositories/code"
	repoRepo "se-school/internal/repositories/repository"
	subRepo "se-school/internal/repositories/subscription"
	repositorySvc "se-school/internal/services/repository"
	subscriptionSvc "se-school/internal/services/subscription"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	_ "se-school/docs/generated" // swagger docs
)

//	@title			GitHub Release Notification API
//	@version		1.0.0
//	@description	GitHub Release Notification API that allows users to subscribe to email notifications about new releases of a chosen GitHub repository.

//	@BasePath	/api

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API key passed in the X-API-Key header

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	logger, err := logging.Init(&cfg.Log)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	zap.ReplaceGlobals(logger)
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Connect(&cfg.Database)
	if err != nil {
		zap.L().Fatal("failed to connect to database", zap.Error(err))
	}

	redisClient, err := redisInfra.Connect(ctx, &cfg.Redis)
	if err != nil {
		zap.L().Fatal("failed to connect to redis", zap.Error(err))
	}

	// Repositories
	subscriptionRepository := subRepo.New(database)
	repositoryRepository := repoRepo.New(database)
	codeRepository := codeRepo.New(database)

	// Integrations
	githubIntegration, err := github.New(&cfg.Github, redisClient)
	if err != nil {
		zap.L().Fatal("failed to initialize github integration", zap.Error(err))
	}

	// Notifications
	notificationService := notifications.New(
		mailer.NewMailerService(&cfg.Mailer),
		templates.New(),
	)

	// Services
	subscriptionService := subscriptionSvc.New(
		cfg,
		subscriptionRepository,
		repositoryRepository,
		codeRepository,
		githubIntegration,
		notificationService,
	)

	repositoryService := repositorySvc.New(
		cfg,
		repositoryRepository,
		subscriptionRepository,
		notificationService,
		githubIntegration,
	)

	// Cron
	cron := cronScheduler.New(ctx, &cfg.Cron, repositoryService)
	cron.Start()

	// Controllers
	subscriptionController := controllers.NewSubscriptionController(subscriptionService)

	// Router. gin.New() (not gin.Default()) is used so request logging and
	// recovery go through our structured zap middlewares instead of Gin's
	// default plain-text logger, keeping the whole log stream JSON.
	r := gin.New()
	controllers.RegisterRoutes(r, subscriptionController, &cfg.Application)

	port := cfg.Application.Port
	if port == "" {
		port = "8080"
	}

	zap.L().Info("starting server", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		zap.L().Fatal("failed to start server", zap.Error(err))
	}
	cron.Stop()
}
