package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	codeUniqueViolation           = "23505"
	codeInvalidTextRepresentation = "22P02"
)

// violatesUnique reports whether err is Postgres refusing a row that would break the unique index named index.
func violatesUnique(err error, index string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == codeUniqueViolation && pgErr.ConstraintName == index
}

// malformedID reports whether err is Postgres refusing an ID that is not a UUID.
func malformedID(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == codeInvalidTextRepresentation
}
