package helpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"se-school/internal/config"
	"se-school/internal/integrations/github"
	"se-school/internal/models"
	"se-school/internal/notifications"
	codeRepo "se-school/internal/repositories/code"
	repoRepo "se-school/internal/repositories/repository"
	subRepo "se-school/internal/repositories/subscription"
	"se-school/internal/services/subscription"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Suite struct {
	T   *testing.T
	Ctx context.Context

	Cfg      *config.Config
	DB       *gorm.DB
	Redis    *redis.Client
	GH       *MSWServer
	Notifier *notifications.NotificationsServiceMock

	Svc      *subscription.Service
	SubRepo  *subRepo.Repository
	RepoRepo *repoRepo.Repository
	CodeRepo *codeRepo.Repository
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

	db := connectDB(t, dsn)
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	truncate(t, db)

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

	notifier := notifications.NewNotificationsServiceMock()
	githubSvc, err := github.New(&cfg.Github, rdb)
	if err != nil {
		t.Fatalf("github integration: %v", err)
	}

	subscriptionsRepo := subRepo.New(db)
	repositoriesRepo := repoRepo.New(db)
	codesRepo := codeRepo.New(db)

	svc := subscription.New(
		cfg,
		subscriptionsRepo,
		repositoriesRepo,
		codesRepo,
		githubSvc,
		notifier,
	)

	s := &Suite{
		T:        t,
		Ctx:      ctx,
		Cfg:      cfg,
		DB:       db,
		Redis:    rdb,
		GH:       gh,
		Notifier: notifier,
		Svc:      svc,
		SubRepo:  subscriptionsRepo,
		RepoRepo: repositoriesRepo,
		CodeRepo: codesRepo,
	}

	t.Cleanup(func() {
		truncate(t, db)
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

func connectDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	var (
		db  *gorm.DB
		err error
	)
	deadline := time.Now().Add(30 * time.Second)
	for {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err == nil {
			if sqlDB, e := db.DB(); e == nil && sqlDB.Ping() == nil {
				return db
			}
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

func migrate(db *gorm.DB) error {
	for _, m := range []models.MigratableModel{
		&models.Subscription{},
		&models.Repository{},
		&models.Code{},
	} {
		if err := m.Migrate(db); err != nil {
			return fmt.Errorf("migrate %T: %w", m, err)
		}
	}
	return nil
}

func truncate(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Codes have a FK reference from subscriptions; order matters even
	// with TRUNCATE CASCADE because of the unique partial index.
	if err := db.Exec(`TRUNCATE TABLE subscriptions, repositories, codes RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
