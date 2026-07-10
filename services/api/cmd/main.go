package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"se-school/internal/config"
	"se-school/internal/controllers"
	cronScheduler "se-school/internal/cron"
	"se-school/internal/infrastructure/db"
	kafkaInfra "se-school/internal/infrastructure/kafka"
	"se-school/internal/infrastructure/logging"
	redisInfra "se-school/internal/infrastructure/redis"
	"se-school/internal/integrations/github"
	"se-school/internal/models/factories/codes"
	"se-school/internal/notifications/grpcclient"
	"se-school/internal/notifications/relay"
	"se-school/internal/notifications/replies"
	codeRepo "se-school/internal/repositories/code"
	outboxRepo "se-school/internal/repositories/outbox"
	repoRepo "se-school/internal/repositories/repository"
	sagaRepo "se-school/internal/repositories/saga"
	subRepo "se-school/internal/repositories/subscription"
	repositorySvc "se-school/internal/services/repository"
	subscriptionSvc "se-school/internal/services/subscription"
	"se-school/internal/uow"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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

	// In JSON (pipeline) mode, switch Gin to release so it stops printing its
	// plain-text "[GIN-debug] ..." banner to stdout — those lines aren't JSON and
	// would otherwise show up in Elasticsearch as failed-to-decode records.
	if cfg.Log.Encoding != "console" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Cancel the root context on SIGINT/SIGTERM so the HTTP server, Kafka
	// readers/writers and the background loops (relay, reply consumer, sweeper)
	// all shut down cleanly when the container is stopped.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect(&cfg.Database)
	if err != nil {
		zap.L().Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	redisClient, err := redisInfra.Connect(ctx, &cfg.Redis)
	if err != nil {
		zap.L().Fatal("failed to connect to redis", zap.Error(err))
	}

	// Repositories
	subscriptionRepository := subRepo.New(database)
	repositoryRepository := repoRepo.New(database)
	codeRepository := codeRepo.New(database)
	sagaRepository := sagaRepo.New(database)
	outboxRepository := outboxRepo.New(database)

	// Unit of work: runs the saga's create step (subscription + codes + saga
	// instance + outbox command) atomically in one transaction.
	transactor := db.NewTransactor(database)
	unitOfWork := uow.New(transactor, subscriptionRepository, codeRepository, sagaRepository, outboxRepository)

	// Domain
	codeFactory := codes.NewFactory()

	// Integrations
	githubIntegration, err := github.New(&cfg.Github, redisClient)
	if err != nil {
		zap.L().Fatal("failed to initialize github integration", zap.Error(err))
	}

	// Kafka: the API produces notification commands onto cfg.Kafka.Topic (drained
	// from the transactional outbox by the relay, and directly by the cron
	// publisher) and consumes saga replies from cfg.Kafka.RepliesTopic.
	// EnsureTopics creates events, DLQ and replies topics up front.
	if err := kafkaInfra.EnsureTopics(ctx, &cfg.Kafka); err != nil {
		zap.L().Fatal("failed to ensure kafka topics", zap.Error(err))
	}
	kafkaWriter := kafkaInfra.NewWriter(&cfg.Kafka)
	defer func() { _ = kafkaWriter.Close() }()

	// Notifier gRPC client: the cron repository-update path calls the notifier
	// synchronously so a failed delivery leaves the subscription's last_seen_tag
	// unadvanced (retried next run). The subscribe-confirmation saga still flows
	// over Kafka (outbox + replies).
	notifierConn, err := grpc.NewClient(cfg.Notifier.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		zap.L().Fatal("failed to create notifier gRPC client",
			zap.String("addr", cfg.Notifier.Addr), zap.Error(err))
	}
	defer func() { _ = notifierConn.Close() }()
	notifier := grpcclient.New(notifierConn, cfg.Notifier.Timeout)

	// Services
	subscriptionService := subscriptionSvc.New(
		cfg.FrontendURL,
		subscriptionRepository,
		repositoryRepository,
		codeRepository,
		codeFactory,
		githubIntegration,
		unitOfWork,
		sagaRepository,
		cfg.Saga.Deadline,
	)

	repositoryService := repositorySvc.New(
		cfg.FrontendURL,
		repositoryRepository,
		subscriptionRepository,
		notifier,
		githubIntegration,
	)

	// Outbox relay: publishes the saga commands persisted in T1 to Kafka
	// (at-least-once), closing the create-then-publish dual-write.
	outboxRelay := relay.New(outboxRepository, kafkaWriter, cfg.Saga.RelayInterval, cfg.Saga.BatchSize)
	go outboxRelay.Run(ctx)

	// Saga reply consumer: applies the notifier's dispatch replies, completing or
	// compensating each saga.
	replyReader := kafkaInfra.NewReplyReader(&cfg.Kafka)
	defer func() { _ = replyReader.Close() }()
	replyConsumer := replies.New(replyReader, subscriptionService)
	go func() {
		if err := replyConsumer.Run(ctx); err != nil {
			zap.L().Error("reply consumer stopped with error", zap.Error(err))
		}
	}()

	// Saga sweeper: compensates sagas whose dispatch reply never arrived by the
	// deadline (notifier down / reply lost).
	go func() {
		ticker := time.NewTicker(cfg.Saga.SweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := subscriptionService.Sweep(ctx, cfg.Saga.BatchSize); err != nil {
					zap.L().Error("saga sweep failed", zap.Error(err))
				}
			}
		}
	}()

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

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		zap.L().Info("starting server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Block until a shutdown signal cancels ctx, then drain gracefully. The
	// background loops observe the same ctx and stop on their own.
	<-ctx.Done()
	zap.L().Info("shutdown signal received")
	cron.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("server shutdown error", zap.Error(err))
	}
	zap.L().Info("server stopped")
}
