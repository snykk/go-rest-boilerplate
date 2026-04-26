//go:build integration

package testenv

import (
	"context"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// StartRedis launches a throwaway Redis container and returns a
// caches.RedisCache pointing at it. Mirrors StartPostgres semantics:
// container is terminated via t.Cleanup, defaultTTL is short so
// expiry-driven tests don't have to wait minutes.
func StartRedis(t *testing.T) caches.RedisCache {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer stopCancel()
		if err := testcontainers.TerminateContainer(c, testcontainers.StopContext(stopCtx)); err != nil {
			t.Logf("terminate redis container: %v", err)
		}
	})

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("redis host: %v", err)
	}
	port, err := c.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("redis port: %v", err)
	}

	addr := host + ":" + port.Port()
	// 1-minute default TTL — long enough that no test should hit it
	// accidentally, short enough that explicit Expire-driven tests
	// don't have to wait for natural expiry.
	rc := caches.NewRedisCache(addr, 0, "", time.Minute)
	t.Cleanup(func() { _ = rc.Close() })
	return rc
}
