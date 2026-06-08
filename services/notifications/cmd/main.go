// Command notifier is the notifications microservice. It subscribes to the
// Redis Pub/Sub channel that the API publishes notification jobs to, renders the
// requested email template and delivers it over SMTP. It owns no database and no
// domain logic — its only contract with the rest of the system is ghnotify/contract.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"ghnotify/notifier/internal/config"
	"ghnotify/notifier/internal/logging"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/templates"
	"ghnotify/notifier/internal/worker"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

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

	// Cancel the root context on SIGINT/SIGTERM so the worker unsubscribes and
	// shuts down cleanly when the container is stopped.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		zap.L().Fatal("failed to connect to redis", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()
	zap.L().Info("connected to redis", zap.String("address", cfg.Redis.Address))

	w := worker.New(
		redisClient,
		templates.New(),
		mailer.NewMailerService(&cfg.Mailer),
	)

	zap.L().Info("notifications service started")
	if err := w.Run(ctx); err != nil {
		zap.L().Fatal("worker stopped with error", zap.Error(err))
	}
	zap.L().Info("notifications service stopped")
}
