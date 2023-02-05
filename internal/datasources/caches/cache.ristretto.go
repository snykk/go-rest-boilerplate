package caches

import (
	ristr "github.com/dgraph-io/ristretto"
)

type RistrettoCache interface {
	Set(key string, value interface{})
	Get(key string) interface{}
	Del(key ...string)
}

type ristrettoCache struct {
	cache *ristr.Cache
}

func NewRistrettoCache() (RistrettoCache, error) {
	cache, err := ristr.NewCache(&ristr.Config{
		BufferItems: 64,
		MaxCost:     1 << 30,
		NumCounters: 1e7,
	})
	if err != nil {
		return nil, err
	}

	return &ristrettoCache{cache: cache}, nil
}

func (cache *ristrettoCache) Set(key string, value interface{}) {
	cache.cache.Set(key, value, 1)
}

func (cache *ristrettoCache) Get(key string) interface{} {
	val, _ := cache.cache.Get(key)

	return val
}

func (cache *ristrettoCache) Del(key ...string) {
	for _, i := range key {
		cache.cache.Del(i)
	}
}
