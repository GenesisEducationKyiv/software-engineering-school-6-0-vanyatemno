// Package dedup is the notifications consumer's idempotency store: the worker
// claims a key per recipient before sending, so each email is delivered at most
// once across redeliveries/republishes.
package dedup

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Deduper interface {
	// Claim atomically marks key as taken. It returns true if the key was newly
	// claimed (the caller should send), or false if it was already claimed (the
	// caller should skip — an earlier delivery already sent this email).
	Claim(ctx context.Context, key string) (bool, error)
	// Release removes a claim so a later retry can re-attempt the send. Used when
	// the send fails after the key was claimed.
	Release(ctx context.Context, key string) error
}

// RedisDeduper backs Deduper with Redis `SET NX EX ttl`. The TTL must be long
// enough to cover any redelivery/republish window, short enough to keep Redis
// from growing unbounded.
type RedisDeduper struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisDeduper(client *redis.Client, ttl time.Duration) *RedisDeduper {
	return &RedisDeduper{client: client, ttl: ttl}
}

func (d *RedisDeduper) Claim(ctx context.Context, key string) (bool, error) {
	// SET key "1" NX EX ttl — sets only if absent. A nil reply (redis.Nil) means
	// the key already existed, i.e. the email was already claimed/sent.
	err := d.client.SetArgs(ctx, key, "1", redis.SetArgs{Mode: "NX", TTL: d.ttl}).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *RedisDeduper) Release(ctx context.Context, key string) error {
	return d.client.Del(ctx, key).Err()
}
