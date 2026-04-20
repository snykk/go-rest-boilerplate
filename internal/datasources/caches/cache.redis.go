package caches

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// defaultOpTimeout bounds every Redis operation so a slow/unreachable
// Redis cannot hang caller goroutines indefinitely.
const defaultOpTimeout = 3 * time.Second

type RedisCache interface {
	Set(ctx context.Context, key string, value interface{}) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Close() error
	Client() *redis.Client
}

type redisCache struct {
	host     string
	db       int
	password string
	expires  time.Duration
	client   *redis.Client
}

func NewRedisCache(host string, db int, password string, expires time.Duration) RedisCache {
	return &redisCache{
		host:     host,
		db:       db,
		password: password,
		expires:  expires,
		client: redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password,
			DB:       db,
		}),
	}
}

// withTimeout derives a bounded context from the caller's ctx so Redis
// operations can never block longer than defaultOpTimeout.
func withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, defaultOpTimeout)
}

func (cache *redisCache) Set(ctx context.Context, key string, value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()
	return cache.client.Set(ctx, key, payload, cache.expires*time.Minute).Err()
}

func (cache *redisCache) Get(ctx context.Context, key string) (string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	val, err := cache.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	var decoded string
	if err := json.Unmarshal([]byte(val), &decoded); err != nil {
		return "", err
	}
	return decoded, nil
}

func (cache *redisCache) Del(ctx context.Context, key string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	return cache.client.Del(ctx, key).Err()
}

func (cache *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	return cache.client.Incr(ctx, key).Result()
}

func (cache *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	return cache.client.Expire(ctx, key, ttl).Err()
}

func (cache *redisCache) Close() error {
	return cache.client.Close()
}

func (cache *redisCache) Client() *redis.Client {
	return cache.client
}
