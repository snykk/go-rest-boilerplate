// Package migrate runs the service's SQL schema migrations with goose. It is apart from package database so
// that a binary which only serves requests does not carry the migration engine.
package migrate

import (
	"database/sql"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// New returns a goose provider for the SQL migrations at the root of files, run on database. Migrations apply in
// version order, and a version older than one already applied still applies, so migrations merged from other
// branches are never skipped.
func New(database *sql.DB, files fs.FS) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, database, files, goose.WithAllowOutofOrder(true))
}
