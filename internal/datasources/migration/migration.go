// Package migration is the testable library behind cmd/migration. The
// CLI in cmd/migration/main.go is now a thin wrapper around this
// package — config loading + flag parsing + db wiring — so the
// idempotency / advisory-lock / per-file-tx behavior can be exercised
// in integration tests without exec'ing a binary.
package migration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// AdvisoryLockID is an arbitrary 64-bit integer used with
// pg_advisory_lock so two migration runners can't apply the same file
// concurrently. Exported so a CI smoke test can inspect lock state.
const AdvisoryLockID = 947328461230

// Runner applies / reverts SQL migration files against a Postgres DB
// while tracking applied state in the schema_migrations table.
type Runner struct {
	db  *sqlx.DB
	dir string
	// log lets tests swap in a no-op sink; nil falls back to the
	// project-wide logger.
	log func(msg string, fields logger.Fields)
}

// New builds a Runner that reads migration files from `dir`.
func New(db *sqlx.DB, dir string) *Runner {
	return &Runner{db: db, dir: dir}
}

// SetLogger overrides the default logger sink. Pass a no-op to mute
// the runner during tests.
func (r *Runner) SetLogger(fn func(string, logger.Fields)) { r.log = fn }

// DB exposes the underlying handle for callers that need to assert
// post-migration state (e.g., integration tests querying
// schema_migrations directly). Production code should not reach for
// this — use the Up/Down methods.
func (r *Runner) DB() *sqlx.DB { return r.db }

func (r *Runner) info(msg string, fields logger.Fields) {
	if r.log != nil {
		r.log(msg, fields)
		return
	}
	logger.Info(msg, fields)
}

// Up applies every *.up.sql file in lexicographic order. Files that
// already appear in schema_migrations are skipped, so reruns are
// idempotent. Each file commits its statement and the
// schema_migrations row in a single transaction.
func (r *Runner) Up(ctx context.Context) error {
	r.info("running migration [up]", logger.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})

	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	return r.withAdvisoryLock(ctx, func() error {
		files, err := r.listFiles("up")
		if err != nil {
			return err
		}
		applied, err := r.loadApplied(ctx)
		if err != nil {
			return err
		}

		for _, file := range files {
			name := filepath.Base(file)
			if applied[name] {
				r.info("skipping already-applied migration", logger.Fields{
					constants.LoggerCategory: constants.LoggerCategoryMigration,
					constants.LoggerFile:     name,
				})
				continue
			}
			r.info("applying migration", logger.Fields{
				constants.LoggerCategory: constants.LoggerCategoryMigration,
				constants.LoggerFile:     name,
			})
			if err := r.applyFile(ctx, file, name, true); err != nil {
				return err
			}
		}
		r.info("migration [up] success", logger.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		return nil
	})
}

// Down applies every *.down.sql file in REVERSE lexicographic order
// (later migrations roll back first). Each successful down deletes the
// matching schema_migrations row.
func (r *Runner) Down(ctx context.Context) error {
	r.info("running migration [down]", logger.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})

	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	return r.withAdvisoryLock(ctx, func() error {
		files, err := r.listFiles("down")
		if err != nil {
			return err
		}
		sort.Sort(sort.Reverse(sort.StringSlice(files)))

		for _, file := range files {
			name := filepath.Base(file)
			r.info("reverting migration", logger.Fields{
				constants.LoggerCategory: constants.LoggerCategoryMigration,
				constants.LoggerFile:     name,
			})
			if err := r.applyFile(ctx, file, deriveUpName(name), false); err != nil {
				return err
			}
		}
		r.info("migration [down] success", logger.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		return nil
	})
}

func (r *Runner) ensureMigrationsTable(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name        TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

// withAdvisoryLock serializes concurrent runners (CI + engineer
// laptop) so only one at a time can mutate the schema.
func (r *Runner) withAdvisoryLock(ctx context.Context, fn func() error) error {
	if _, err := r.db.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, AdvisoryLockID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		if _, err := r.db.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, AdvisoryLockID); err != nil {
			logger.Error("failed to release migration advisory lock", logger.Fields{
				constants.LoggerCategory: constants.LoggerCategoryMigration,
				"error":                  err.Error(),
			})
		}
	}()
	return fn()
}

func (r *Runner) listFiles(action string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(r.dir, fmt.Sprintf("*.%s.sql", action)))
	if err != nil {
		return nil, errors.New("glob migration files")
	}
	sort.Strings(files)
	return files, nil
}

func (r *Runner) loadApplied(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

// applyFile runs a migration SQL file inside a transaction together
// with the schema_migrations bookkeeping write — so a crash mid-file
// leaves no partial record.
func (r *Runner) applyFile(ctx context.Context, file, upName string, isUp bool) error {
	// #nosec G304 — file paths come from the developer-controlled
	// migrations directory, not request input.
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(data)); err != nil {
		return fmt.Errorf("exec %s: %w", filepath.Base(file), err)
	}

	if isUp {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(name) VALUES ($1) ON CONFLICT DO NOTHING`, upName); err != nil {
			return fmt.Errorf("record migration %s: %w", upName, err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE name = $1`, upName); err != nil {
			return fmt.Errorf("forget migration %s: %w", upName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", filepath.Base(file), err)
	}
	return nil
}

// deriveUpName converts a "*.down.sql" filename into its "*.up.sql"
// counterpart, which is how migrations are keyed in schema_migrations.
func deriveUpName(downName string) string {
	const suffix = ".down.sql"
	if len(downName) > len(suffix) && downName[len(downName)-len(suffix):] == suffix {
		return downName[:len(downName)-len(suffix)] + ".up.sql"
	}
	return downName
}
