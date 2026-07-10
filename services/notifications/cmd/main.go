// Command notifier is the notifications microservice. It delivers emails over
// two transports that share one at-most-once delivery core (internal/dispatch):
// it consumes the Kafka topic the API publishes saga commands to (confirmation
// emails, replying with the outcome), and it serves a synchronous gRPC API the
// API calls for repository-update alerts. Postgres records durable per-delivery
// state (SENDING → SENT/FAILED) and Redis is a fast idempotency store, so every
// email is sent at most once. Its only wire contract with the rest of the system
// is ghnotify/contract.
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os/signal"
	"syscall"

	"ghnotify/contract/notifierpb"
	"ghnotify/notifier/internal/config"
	"ghnotify/notifier/internal/dedup"
	"ghnotify/notifier/internal/dispatch"
	"ghnotify/notifier/internal/grpcserver"
	dbInfra "ghnotify/notifier/internal/infrastructure/db"
	kafkaInfra "ghnotify/notifier/internal/infrastructure/kafka"
	"ghnotify/notifier/internal/logging"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/repositories/delivery"
	"ghnotify/notifier/internal/templates"
	"ghnotify/notifier/internal/worker"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	// Cancel the root context on SIGINT/SIGTERM so the worker stops consuming and
	// shuts down cleanly when the container is stopped.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Redis is the idempotency store: the worker claims a marker per recipient
	// before sending so each email goes out at most once even on redelivery.
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

	// Postgres is the notifier's durable participant state: it records each
	// confirmation email's dispatch outcome (SENDING → SENT/FAILED), which the
	// saga reply is derived from.
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

	// Delivery dependencies, shared by both transports.
	deduper := dedup.NewRedisDeduper(redisClient, cfg.DedupTTL)
	templateService := templates.New()
	mailerService := mailer.NewMailerService(&cfg.Mailer)

	// dispatcher is the at-most-once delivery core the gRPC server uses; the Kafka
	// worker builds an equivalent one from the same stores.
	dispatcher := dispatch.New(deliveryRepository, deduper, templateService, mailerService, cfg.Kafka.MaxRetries)

	w := worker.New(
		reader,
		dlqWriter,
		replyWriter,
		deduper,
		deliveryRepository,
		templateService,
		mailerService,
		cfg.Kafka.MaxRetries,
	)

	// gRPC server: the API calls Notify synchronously for repository-update
	// alerts. It runs alongside the Kafka worker and is drained on shutdown.
	grpcServer := grpc.NewServer()
	notifierpb.RegisterNotifierServer(grpcServer, grpcserver.New(dispatcher))
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", cfg.GRPC.Port)
	if err != nil {
		zap.L().Fatal("failed to listen for gRPC", zap.String("port", cfg.GRPC.Port), zap.Error(err))
	}
	go func() {
		zap.L().Info("gRPC server listening", zap.String("address", cfg.GRPC.Port))
		if serveErr := grpcServer.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			zap.L().Error("gRPC server stopped with error", zap.Error(serveErr))
		}
	}()

	zap.L().Info("notifications service started")
	if err := w.Run(ctx); err != nil {
		zap.L().Fatal("worker stopped with error", zap.Error(err))
	}

	// Root context cancelled (SIGINT/SIGTERM): stop accepting new RPCs and drain
	// any in-flight ones before the deferred resource closes fire.
	zap.L().Info("shutting down gRPC server")
	grpcServer.GracefulStop()
	zap.L().Info("notifications service stopped")
}
