// Package migrations embeds the SQL schema migrations of every module. goose applies them in version order.
package migrations

import "embed"

// FS holds every .sql migration in this directory.
//
//go:embed *.sql
var FS embed.FS
