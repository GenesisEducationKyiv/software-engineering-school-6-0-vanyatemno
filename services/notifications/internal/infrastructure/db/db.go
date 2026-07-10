// Package db connects the notifications service to its own Postgres and runs its
// embedded migrations.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"ghnotify/notifier/internal/config"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	// pgx5 migrate driver — registered via init() under the "pgx5" scheme.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect also runs pending migrations. Callers own the returned pool and must
// Close it on shutdown.
func Connect(cfg *config.Database) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
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

	if err := runMigrations(poolCfg); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return pool, nil
}

func runMigrations(poolCfg *pgxpool.Config) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("open migrations fs: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, toMigrateURL(poolCfg))
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func toMigrateURL(cfg *pgxpool.Config) string {
	cc := cfg.ConnConfig
	u := &url.URL{
		Scheme: "pgx5",
		User:   url.UserPassword(cc.User, cc.Password),
		Host:   net.JoinHostPort(cc.Host, strconv.Itoa(int(cc.Port))),
		Path:   "/" + cc.Database,
	}

	q := u.Query()
	for k, v := range cc.RuntimeParams {
		q.Set(k, v)
	}
	if _, set := q["sslmode"]; !set {
		if cc.TLSConfig == nil {
			q.Set("sslmode", "disable")
		} else {
			q.Set("sslmode", "require")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
