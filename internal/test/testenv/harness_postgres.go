//go:build integration

// Package testenv contains the integration-test harness — disposable
// Postgres and Redis containers spun up via testcontainers-go and
// torn down via t.Cleanup, plus the migration loader that primes a
// fresh schema on each container.
//
// The whole package is gated by the `integration` build tag, so it
// is excluded from `go build ./...` and from the default `go test`.
// Production binaries never link testcontainers-go, and unit tests
// never need Docker.
//
// To run integration tests: `make test-integration` (requires Docker).
package testenv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPostgresEmpty launches a throwaway Postgres container, installs
// the uuid-ossp extension, and returns a *sqlx.DB pointing at it. No
// migrations are applied. Use this when the test itself drives schema
// changes (e.g., the migration runner's own integration test).
func StartPostgresEmpty(t *testing.T) *sqlx.DB {
	t.Helper()
	return startPostgres(t, false)
}

// StartPostgres launches a throwaway Postgres container, applies every
// .up.sql migration in cmd/migration/migrations in lexicographic order,
// and returns a connected *sqlx.DB. The container is terminated by
// t.Cleanup so each test gets a clean slate; nothing leaks between
// tests in the same package.
func StartPostgres(t *testing.T) *sqlx.DB {
	t.Helper()
	return startPostgres(t, true)
}

func startPostgres(t *testing.T, runMigrations bool) *sqlx.DB {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const (
		dbName = "boilerplate_test"
		dbUser = "test"
		dbPass = "test"
	)

	c, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		// Use a fresh ctx — t.Cleanup runs after the test ctx is done.
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer stopCancel()
		if err := testcontainers.TerminateContainer(c, testcontainers.StopContext(stopCtx)); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// uuid_generate_v4() is used by the users migration.
	if _, err := db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`); err != nil {
		t.Fatalf("install uuid-ossp: %v", err)
	}

	if err := applyMigrations(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

// applyMigrations runs every .up.sql file in lexicographic order. Kept
// inline in the harness rather than reusing cmd/migration so the
// integration test stays decoupled from the runner's transaction /
// schema_migrations bookkeeping.
func applyMigrations(ctx context.Context, db *sqlx.DB) error {
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".sql" && len(name) > len(".up.sql") && name[len(name)-len(".up.sql"):] == ".up.sql" {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, name := range files {
		full := filepath.Join(dir, name)
		// #nosec G304 — `full` is built from a developer-controlled
		// migrations directory, not request input.
		data, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
	}
	return nil
}

// migrationsDir resolves the project-root migrations directory by
// walking up from this source file until it finds go.mod, so the
// harness keeps working when its package is moved inside internal/.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "cmd", "migration", "migrations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod —
			// return a path that will fail loudly downstream rather
			// than silently picking up a wrong directory.
			return filepath.Join("cmd", "migration", "migrations")
		}
		dir = parent
	}
}
