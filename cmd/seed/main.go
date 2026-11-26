// Command seed fills a development database with sample content: a verified editor and a verified reader, two
// published articles and a draft, and two comments by the reader. It signs in as those accounts and calls the
// application's handler in process, so every rule of the API applies, and it skips what already exists, so it
// can run twice. Run the migrations first.
//
// It reads the same configuration as the server, from environment variables and an optional ./.env file, and
// SEED_EDITOR_EMAIL, SEED_EDITOR_PASSWORD, SEED_READER_EMAIL and SEED_READER_PASSWORD for the accounts. The
// defaults of those are for local development only, and the command refuses to run with APP_ENV=production.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/snykk/go-rest-boilerplate/internal/app"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	source, err := config.LoadSource()
	if err != nil {
		return err
	}
	return execute(ctx, source)
}

// execute reads the settings from source and, unless they are invalid or APP_ENV is production, builds what Seed
// needs as the server does and runs it. No telemetry is pushed and the mail is recorded in memory, not sent,
// because creating the accounts and signing in send none.
func execute(ctx context.Context, source config.Source) error {
	appConfig, err := app.LoadConfig(source)
	if err != nil {
		return err
	}
	if appConfig.Platform.Production() {
		return errors.New("seed data is for development only; APP_ENV is production")
	}

	level := new(slog.LevelVar)
	level.Set(appConfig.Platform.Log.Level)
	log, err := logger.New(os.Stdout, logger.Options{
		Level:  level,
		Format: appConfig.Platform.Log.Format,
		Source: appConfig.Platform.Log.Source,
	})
	if err != nil {
		return err
	}

	providers, err := telemetry.New(ctx, telemetry.Config{
		Service:     appConfig.Platform.Service,
		Environment: appConfig.Platform.Env,
	})
	if err != nil {
		return err
	}
	defer func() { _ = providers.Shutdown(context.WithoutCancel(ctx)) }()

	pool, err := database.Open(ctx, appConfig.Platform.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	return Seed(ctx, appConfig, app.Deps{
		Log:       log,
		Level:     level,
		Pool:      pool,
		Telemetry: providers,
		Mail:      &mailer.Recorder{},
	}, loadAccounts(source))
}

// loadAccounts reads the accounts from SEED_EDITOR_EMAIL (editor@example.com), SEED_EDITOR_PASSWORD
// (editor-password), SEED_READER_EMAIL (reader@example.com) and SEED_READER_PASSWORD (reader-password).
func loadAccounts(source config.Source) Accounts {
	reader := config.NewReader(source)
	return Accounts{
		EditorEmail:    reader.String("SEED_EDITOR_EMAIL", "editor@example.com"),
		EditorPassword: reader.String("SEED_EDITOR_PASSWORD", "editor-password"),
		ReaderEmail:    reader.String("SEED_READER_EMAIL", "reader@example.com"),
		ReaderPassword: reader.String("SEED_READER_PASSWORD", "reader-password"),
	}
}
