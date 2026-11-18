// Command migrate applies, rolls back, lists and creates the SQL schema migrations.
//
// Usage:
//
//	migrate up | down | status | create <name>
//
// up applies every pending migration, down rolls back the latest one, and status lists them all; these read
// DATABASE_URL from the environment or ./.env. create writes a new, empty migration to ./migrations and needs
// no database, so it runs from the repository root.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
	"github.com/snykk/go-rest-boilerplate/internal/platform/migrate"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

const usage = "usage: migrate up | down | status | create <name>"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 2 && args[0] == "create" {
		return goose.Create(nil, "migrations", args[1], "sql")
	}
	var subcommand func(context.Context, *goose.Provider) error
	if len(args) == 1 {
		subcommand = command(args[0])
	}
	if subcommand == nil {
		return errors.New(usage)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	settings, err := config.LoadDatabase()
	if err != nil {
		return err
	}

	database, err := sql.Open("pgx", settings.URL)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	provider, err := migrate.New(database, migrations.FS)
	if err != nil {
		return err
	}

	return subcommand(ctx, provider)
}

func command(name string) func(context.Context, *goose.Provider) error {
	switch name {
	case "up":
		return up
	case "down":
		return down
	case "status":
		return status
	}
	return nil
}

func up(ctx context.Context, provider *goose.Provider) error {
	results, err := provider.Up(ctx)
	for _, result := range results {
		fmt.Println(result)
	}
	if err == nil && len(results) == 0 {
		fmt.Println("no pending migrations")
	}
	return err
}

func down(ctx context.Context, provider *goose.Provider) error {
	result, err := provider.Down(ctx)
	if errors.Is(err, goose.ErrNoNextVersion) {
		fmt.Println("no migration to roll back")
		return nil
	}
	if result != nil {
		fmt.Println(result)
	}
	return err
}

func status(ctx context.Context, provider *goose.Provider) error {
	statuses, err := provider.Status(ctx)
	if err != nil {
		return err
	}
	for _, migration := range statuses {
		applied := "-"
		if migration.State == goose.StateApplied {
			applied = migration.AppliedAt.UTC().Format(time.RFC3339)
		}
		fmt.Printf("%-8s %-20s %s\n", migration.State, applied, migration.Source.Path)
	}
	return nil
}
