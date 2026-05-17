package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"se-school/internal/config"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	// pgx5 migrate driver — registered via init() under the "pgx5" scheme.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect opens a pgxpool to the configured Postgres instance, runs any
// pending migrations embedded under migrations/, and returns the pool.
// Callers own the pool and must Close it when shutting down.
func Connect(cfg *config.Database) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DNS)
	if err != nil {
		return nil, fmt.Errorf("parse db dsn: %w", err)
	}

	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := runMigrations(cfg.DNS); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return pool, nil
}

func runMigrations(dsn string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("open migrations fs: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, toMigrateDSN(dsn))
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// toMigrateDSN re-prefixes a URL-form Postgres DSN with the "pgx5" scheme that
// the golang-migrate pgx/v5 driver expects. Keyword-form DSNs are unsupported
// by the migrate driver and are passed through unchanged.
func toMigrateDSN(dsn string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(dsn, prefix) {
			return "pgx5://" + strings.TrimPrefix(dsn, prefix)
		}
	}
	return dsn
}
