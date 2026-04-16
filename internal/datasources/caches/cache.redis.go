package caches

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache interface {
	Set(key string, value interface{}) error
	Get(key string) (string, error)
	Del(key string) error
	Close() error
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

func (cache *redisCache) Set(key string, value interface{}) error {
	json, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return cache.client.Set(context.Background(), key, json, cache.expires*time.Minute).Err()
}

func (cache *redisCache) Get(key string) (email string, err error) {
	val, err := cache.client.Get(context.Background(), key).Result()
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(val), &email)
	return email, err
}

func (cache *redisCache) Del(key string) error {
	return cache.client.Del(context.Background(), key).Err()
}

func (cache *redisCache) Close() error {
	return cache.client.Close()
}
