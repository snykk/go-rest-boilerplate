package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier runs SQL. It is either a pool, on which every call stands alone, or a transaction, whose calls
// commit or roll back together. A gateway that takes a Querier runs unchanged on either, so a unit of work
// builds its repositories on one transaction.
type Querier interface {
	// Exec runs sql, a statement that returns no rows, with args.
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	// Query runs sql with args and returns its rows, which the caller closes.
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	// QueryRow runs sql with args and returns its first row; the row's Scan returns pgx.ErrNoRows when there
	// is none.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// Begin starts a transaction on a pool, or a savepoint inside a transaction.
	Begin(ctx context.Context) (pgx.Tx, error)
}

var (
	_ Querier = (*pgxpool.Pool)(nil)
	_ Querier = pgx.Tx(nil)
)
