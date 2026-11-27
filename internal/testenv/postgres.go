//go:build integration

package testenv

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
	"github.com/snykk/go-rest-boilerplate/internal/platform/migrate"
)

var (
	startOnce sync.Once
	container *postgres.PostgresContainer
	adminURL  string
	startErr  error
	databases atomic.Int64
)

// Main runs the package's tests, terminates the Postgres container they started, and exits with their
// result. Every package whose tests call Postgres calls Main from its TestMain.
func Main(m *testing.M) {
	code := m.Run()
	if container != nil {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("testenv: terminate the postgres container: %v", err)
		}
	}
	os.Exit(code)
}

// Postgres returns a pool on a new, empty database of the test's own inside the test binary's shared
// Postgres container. It applies files's migrations first when files is not nil, and drops the database when
// the test ends.
func Postgres(t *testing.T, files fs.FS) *pgxpool.Pool {
	t.Helper()
	startOnce.Do(start)
	if startErr != nil {
		t.Fatalf("testenv: start postgres: %v", startErr)
	}

	ctx := context.Background()
	name := fmt.Sprintf("it_%d_%d", os.Getpid(), databases.Add(1))
	if err := admin(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("testenv: create database %s: %v", name, err)
	}
	t.Cleanup(func() {
		if err := admin(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("testenv: drop database %s: %v", name, err)
		}
	})

	databaseURL, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("testenv: parse the admin URL: %v", err)
	}
	databaseURL.Path = "/" + name
	dsn := databaseURL.String()
	if files != nil {
		applyMigrations(t, dsn, files)
	}

	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("testenv: open %s: %v", name, err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func start() {
	ctx := context.Background()
	container, startErr = postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("app"),
		postgres.WithUsername("app"),
		postgres.WithPassword("app"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if startErr != nil {
		return
	}
	adminURL, startErr = container.ConnectionString(ctx, "sslmode=disable")
}

func admin(ctx context.Context, statement string) error {
	connection, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close(ctx) }()

	_, err = connection.Exec(ctx, statement)
	return err
}

func applyMigrations(t *testing.T, dsn string, files fs.FS) {
	t.Helper()
	connection, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("testenv: open a migration connection: %v", err)
	}
	defer func() { _ = connection.Close() }()

	provider, err := migrate.New(connection, files)
	if err != nil {
		t.Fatalf("testenv: load migrations: %v", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("testenv: apply migrations: %v", err)
	}
}
