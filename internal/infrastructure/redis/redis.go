package redis

import (
	"context"
	"se-school/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func Connect(ctx context.Context, cfg *config.Redis) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	zap.L().Info("connected to redis", zap.String("address", cfg.Address))

	return client, nil
}
