package helpers

import (
	"context"
	"os"
	"testing"
	"time"

	"se-school/internal/config"
	dbinfra "se-school/internal/infrastructure/db"
	"se-school/internal/integrations/github"
	codesFactory "se-school/internal/models/factories/codes"
	codeRepo "se-school/internal/repositories/code"
	outboxRepo "se-school/internal/repositories/outbox"
	repoRepo "se-school/internal/repositories/repository"
	sagaRepo "se-school/internal/repositories/saga"
	subRepo "se-school/internal/repositories/subscription"
	"se-school/internal/services/subscription"
	"se-school/internal/uow"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Suite struct {
	T   *testing.T
	Ctx context.Context

	Cfg   *config.Config
	DB    *pgxpool.Pool
	Redis *redis.Client
	GH    *MSWServer

	Svc        *subscription.Service
	SubRepo    *subRepo.Repository
	RepoRepo   *repoRepo.Repository
	CodeRepo   *codeRepo.Repository
	SagaRepo   *sagaRepo.Repository
	OutboxRepo *outboxRepo.Repository
	Factory    *codesFactory.Factory
}

// NewSuite spins up a fully wired subscription service backed by real
// Postgres + Redis (addresses come from env), with the GitHub client
// pointed at an in-process MSW mock and a mock notifier. The harness
// resets DB state and Redis keys so tests are order-independent.
func NewSuite(t *testing.T) *Suite {
	t.Helper()

	dsn := requireEnv(t, "TEST_DB_DSN")
	redisAddr := requireEnv(t, "TEST_REDIS_ADDR")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	pool := connectDB(t, dsn)
	t.Cleanup(pool.Close)
	truncate(t, pool)

	rdb := connectRedis(t, ctx, redisAddr)
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("redis flushdb: %v", err)
	}

	gh := NewMSWServer(t)
	cfg := &config.Config{
		Github: config.Github{
			Token:   "test-token",
			BaseURL: gh.URL(),
		},
	}

	githubSvc, err := github.New(&cfg.Github, rdb)
	if err != nil {
		t.Fatalf("github integration: %v", err)
	}

	factory := codesFactory.NewFactory()
	subscriptionsRepo := subRepo.New(pool)
	repositoriesRepo := repoRepo.New(pool)
	codesRepo := codeRepo.New(pool)
	sagasRepo := sagaRepo.New(pool)
	outboxRepository := outboxRepo.New(pool)

	transactor := dbinfra.NewTransactor(pool)
	unitOfWork := uow.New(transactor, subscriptionsRepo, codesRepo, sagasRepo, outboxRepository)

	svc := subscription.New(
		cfg.FrontendURL,
		subscriptionsRepo,
		repositoriesRepo,
		codesRepo,
		factory,
		githubSvc,
		unitOfWork,
		sagasRepo,
		2*time.Minute,
	)

	s := &Suite{
		T:          t,
		Ctx:        ctx,
		Cfg:        cfg,
		DB:         pool,
		Redis:      rdb,
		GH:         gh,
		Svc:        svc,
		SubRepo:    subscriptionsRepo,
		RepoRepo:   repositoriesRepo,
		CodeRepo:   codesRepo,
		SagaRepo:   sagasRepo,
		OutboxRepo: outboxRepository,
		Factory:    factory,
	}

	t.Cleanup(func() {
		truncate(t, pool)
		_ = rdb.FlushDB(ctx).Err()
	})
	return s
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("missing env %s", key)
	}
	return v
}

// connectDB opens a pgxpool through the production db.Connect helper, which
// also applies the embedded migrations. It retries briefly so the suite is
// resilient to Postgres still warming up inside docker-compose.
func connectDB(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		pool, err := dbinfra.Connect(&config.Database{DNS: dsn})
		if err == nil {
			return pool
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect postgres: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func connectRedis(t *testing.T, ctx context.Context, addr string) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: addr})
	deadline := time.Now().Add(15 * time.Second)
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return client
		} else if time.Now().After(deadline) {
			t.Fatalf("connect redis: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE TABLE subscriptions, repositories, codes, saga_instances, outbox RESTART IDENTITY CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
