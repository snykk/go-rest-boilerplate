package caches

import (
	ristr "github.com/dgraph-io/ristretto"
)

type RistrettoCache interface {
	// Set stores value under key with cost 1. Writes are async — the
	// value may not be visible to a subsequent Get immediately.
	Set(key string, value interface{})
	// Get returns the cached value, or nil on miss / type-mismatch.
	// Callers must type-assert.
	Get(key string) interface{}
	// Del removes one or more keys. Missing keys are not an error.
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
