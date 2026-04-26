//go:build integration

package migration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/datasources/migration"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeMigration is a tiny helper that drops a synthetic migration
// pair into dir so the test doesn't depend on the real cmd/migration
// files (which evolve and would couple the test to schema changes).
func writeMigration(t *testing.T, dir, num, body, downBody string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, num+"_test.up.sql"), []byte(body), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, num+"_test.down.sql"), []byte(downBody), 0o600))
}

func newRunner(t *testing.T) (*migration.Runner, string) {
	t.Helper()
	db := testenv.StartPostgresEmpty(t)
	dir := t.TempDir()
	r := migration.New(db, dir)
	// Mute the runner's chatter — testing.T already shows what failed.
	r.SetLogger(func(string, logrus.Fields) {})
	return r, dir
}

func TestRunner_UpIsIdempotent(t *testing.T) {
	r, dir := newRunner(t)
	ctx := context.Background()

	writeMigration(t, dir, "1",
		`CREATE TABLE widgets (id SERIAL PRIMARY KEY, name TEXT NOT NULL);`,
		`DROP TABLE widgets;`,
	)

	require.NoError(t, r.Up(ctx))
	// Second invocation must be a no-op — schema_migrations short-
	// circuits the file. Without the tracking table, this would error
	// on "relation widgets already exists".
	require.NoError(t, r.Up(ctx), "second Up should be a no-op")
}

func TestRunner_DownThenUpRoundTrip(t *testing.T) {
	r, dir := newRunner(t)
	ctx := context.Background()

	writeMigration(t, dir, "1",
		`CREATE TABLE widgets (id SERIAL PRIMARY KEY);`,
		`DROP TABLE widgets;`,
	)

	require.NoError(t, r.Up(ctx))
	require.NoError(t, r.Down(ctx))
	// After Down, the row in schema_migrations must be gone, so a
	// follow-up Up applies the migration cleanly.
	require.NoError(t, r.Up(ctx))
}

func TestRunner_PartialFailureRollsBack(t *testing.T) {
	r, dir := newRunner(t)
	ctx := context.Background()

	// The DDL succeeds; the second statement is intentionally invalid
	// SQL to force a mid-file failure. With the per-file transaction
	// the table create AND the schema_migrations bookkeeping must
	// both roll back — leaving the database exactly as it was.
	writeMigration(t, dir, "1",
		`CREATE TABLE half_done (id SERIAL PRIMARY KEY); SELECT this_function_does_not_exist();`,
		`DROP TABLE half_done;`,
	)

	err := r.Up(ctx)
	require.Error(t, err, "Up must surface the SQL error")

	// The failed migration's name must NOT appear in schema_migrations.
	db := r.DB()
	var count int
	require.NoError(t, db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM schema_migrations WHERE name = $1`, "1_test.up.sql"))
	assert.Equal(t, 0, count, "schema_migrations must not record a failed migration")

	// And the table itself must not exist (rollback caught the DDL).
	var exists bool
	require.NoError(t, db.GetContext(ctx, &exists,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'half_done')`))
	assert.False(t, exists, "half-applied DDL must roll back with the tx")
}

func TestRunner_AppliesMultipleFilesInOrder(t *testing.T) {
	r, dir := newRunner(t)
	ctx := context.Background()

	// Two migrations with a dependency: #2 references the table
	// created by #1. If files were applied in the wrong order, #2
	// would fail.
	writeMigration(t, dir, "1",
		`CREATE TABLE a (id SERIAL PRIMARY KEY);`,
		`DROP TABLE a;`,
	)
	writeMigration(t, dir, "2",
		`CREATE TABLE b (id SERIAL PRIMARY KEY, a_id INTEGER REFERENCES a(id));`,
		`DROP TABLE b;`,
	)

	require.NoError(t, r.Up(ctx))

	// Both schema_migrations rows should be present.
	db := r.DB()
	var count int
	require.NoError(t, db.GetContext(ctx, &count, `SELECT COUNT(*) FROM schema_migrations`))
	assert.Equal(t, 2, count)
}
