// Command notifier is the notifications microservice: it consumes the Kafka
// topic the API publishes jobs to, renders the email template and delivers it
// over SMTP. Redis guards at-most-once delivery; Postgres holds the durable
// saga-participant state.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"ghnotify/notifier/internal/config"
	"ghnotify/notifier/internal/dedup"
	dbInfra "ghnotify/notifier/internal/infrastructure/db"
	kafkaInfra "ghnotify/notifier/internal/infrastructure/kafka"
	"ghnotify/notifier/internal/logging"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/repositories/delivery"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Redis is the per-recipient idempotency store guarding at-most-once delivery.
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

	// Postgres holds the durable delivery state the saga reply is derived from.
	database, err := dbInfra.Connect(&cfg.Database)
	if err != nil {
		zap.L().Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close()
	zap.L().Info("connected to database")
	deliveryRepository := delivery.New(database)

	reader := kafkaInfra.NewReader(&cfg.Kafka)
	defer func() { _ = reader.Close() }()
	dlqWriter := kafkaInfra.NewDLQWriter(&cfg.Kafka)
	defer func() { _ = dlqWriter.Close() }()
	replyWriter := kafkaInfra.NewReplyWriter(&cfg.Kafka)
	defer func() { _ = replyWriter.Close() }()

	w := worker.New(
		reader,
		dlqWriter,
		replyWriter,
		dedup.NewRedisDeduper(redisClient, cfg.DedupTTL),
		deliveryRepository,
		templates.New(),
		mailer.NewMailerService(&cfg.Mailer),
		cfg.Kafka.MaxRetries,
	)

	zap.L().Info("notifications service started")
	if err := w.Run(ctx); err != nil {
		zap.L().Fatal("worker stopped with error", zap.Error(err))
	}
	zap.L().Info("notifications service stopped")
}
