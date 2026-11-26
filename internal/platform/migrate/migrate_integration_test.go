//go:build integration

package migrate_test

import (
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/snykk/go-rest-boilerplate/internal/platform/migrate"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
)

// New returns a migrator that applies a migration merged from another branch even when its version is older than one already
// applied.
func TestNew(t *testing.T) {
	pool := testenv.Postgres(t, nil)
	database := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { _ = database.Close() })
	up := func(table string) []byte {
		return []byte("-- +goose Up\nCREATE TABLE " + table + " (id int);\n-- +goose Down\nDROP TABLE " + table + ";\n")
	}
	files := fstest.MapFS{
		"20260101000000_test_a.sql": {Data: up("a")},
		"20260103000000_test_c.sql": {Data: up("c")},
	}
	migrateUp := func() {
		t.Helper()
		provider, err := migrate.New(database, files)
		if err != nil {
			t.Fatalf("New = %v", err)
		}
		if _, err := provider.Up(t.Context()); err != nil {
			t.Fatalf("Up = %v", err)
		}
	}

	migrateUp()
	files["20260102000000_test_b.sql"] = &fstest.MapFile{Data: up("b")}
	migrateUp()

	var count int
	if err := pool.QueryRow(t.Context(),
		`SELECT count(*) FROM information_schema.tables WHERE table_name IN ('a','b','c')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("tables a, b, c present = %d, want 3", count)
	}
}
