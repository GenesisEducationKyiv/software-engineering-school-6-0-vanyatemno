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

// toMigrateURL renders the already-parsed pgx connection details as a pgx5://
// URL for the golang-migrate pgx/v5 driver. This works regardless of whether
// the original DSN was URL-form (postgres://…) or keyword-form (host=… user=…).
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
	// pgx parses `sslmode` into TLSConfig rather than RuntimeParams, so it gets
	// dropped above. Reconstruct sslmode from TLSConfig presence so the migrate
	// driver opens the connection the same way pgxpool does.
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
