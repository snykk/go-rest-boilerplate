package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/utils"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

const (
	dir = "cmd/migration/migrations"

	// advisoryLockID is an arbitrary 64-bit integer used with
	// pg_advisory_lock so two migration runners can't apply the same
	// file concurrently. Any constant works as long as all runners
	// agree on it.
	advisoryLockID = 947328461230
)

var (
	up   bool
	down bool
)

func init() {
	if err := config.InitializeAppConfig(); err != nil {
		logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryConfig})
	}
	logger.Info("configuration loaded", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryConfig})
}

func main() {
	flag.BoolVar(&up, "up", false, "apply new tables, columns, or other structures")
	flag.BoolVar(&down, "down", false, "drop tables, columns, or other structures")
	flag.Parse()

	db, err := utils.SetupPostgresConnection()
	if err != nil {
		logger.Panic(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
	}
	defer db.Close()

	if up {
		if err := migrateUp(db); err != nil {
			logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		}
	}

	if down {
		if err := migrateDown(db); err != nil {
			logger.Fatal(err.Error(), logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		}
	}
}

// ensureMigrationsTable creates the tracking table used to skip
// already-applied migrations on subsequent runs.
func ensureMigrationsTable(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name        TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

// withAdvisoryLock serializes concurrent runners (CI + engineer laptop)
// so only one at a time can mutate the schema.
func withAdvisoryLock(db *sqlx.DB, fn func() error) error {
	if _, err := db.Exec(`SELECT pg_advisory_lock($1)`, advisoryLockID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		if _, err := db.Exec(`SELECT pg_advisory_unlock($1)`, advisoryLockID); err != nil {
			logger.Error("failed to release migration advisory lock", logrus.Fields{
				constants.LoggerCategory: constants.LoggerCategoryMigration,
				"error":                  err.Error(),
			})
		}
	}()
	return fn()
}

func migrateUp(db *sqlx.DB) error {
	logger.Info("running migration [up]", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})

	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	return withAdvisoryLock(db, func() error {
		files, err := listMigrationFiles("up")
		if err != nil {
			return err
		}

		applied, err := loadApplied(db)
		if err != nil {
			return err
		}

		for _, file := range files {
			name := filepath.Base(file)
			if applied[name] {
				logger.Info("skipping already-applied migration", logrus.Fields{
					constants.LoggerCategory: constants.LoggerCategoryMigration,
					constants.LoggerFile:     name,
				})
				continue
			}

			logger.Info("applying migration", logrus.Fields{
				constants.LoggerCategory: constants.LoggerCategoryMigration,
				constants.LoggerFile:     name,
			})
			if err := applyFile(db, file, name, true); err != nil {
				return err
			}
		}

		logger.Info("migration [up] success", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		return nil
	})
}

func migrateDown(db *sqlx.DB) error {
	logger.Info("running migration [down]", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})

	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	return withAdvisoryLock(db, func() error {
		files, err := listMigrationFiles("down")
		if err != nil {
			return err
		}
		// Apply down files in reverse so later migrations roll back first.
		sort.Sort(sort.Reverse(sort.StringSlice(files)))

		for _, file := range files {
			name := filepath.Base(file)
			logger.Info("reverting migration", logrus.Fields{
				constants.LoggerCategory: constants.LoggerCategoryMigration,
				constants.LoggerFile:     name,
			})
			if err := applyFile(db, file, deriveUpName(name), false); err != nil {
				return err
			}
		}

		logger.Info("migration [down] success", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryMigration})
		return nil
	})
}

// listMigrationFiles returns migration filenames for the given action,
// sorted lexicographically.
func listMigrationFiles(action string) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(cwd, dir, fmt.Sprintf("*.%s.sql", action)))
	if err != nil {
		return nil, errors.New("glob migration files")
	}
	sort.Strings(files)
	return files, nil
}

func loadApplied(db *sqlx.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT name FROM schema_migrations`)
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

// applyFile runs a migration SQL file inside a transaction and
// updates the schema_migrations tracking table atomically with the
// statement execution — so a crash mid-file leaves no partial record.
func applyFile(db *sqlx.DB, file, upName string, isUp bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(string(data)); err != nil {
		return fmt.Errorf("exec %s: %w", filepath.Base(file), err)
	}

	if isUp {
		if _, err := tx.Exec(`INSERT INTO schema_migrations(name) VALUES ($1) ON CONFLICT DO NOTHING`, upName); err != nil {
			return fmt.Errorf("record migration %s: %w", upName, err)
		}
	} else {
		if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE name = $1`, upName); err != nil {
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
	if len(downName) > len(".down.sql") && downName[len(downName)-len(".down.sql"):] == ".down.sql" {
		return downName[:len(downName)-len(".down.sql")] + ".up.sql"
	}
	return downName
}
