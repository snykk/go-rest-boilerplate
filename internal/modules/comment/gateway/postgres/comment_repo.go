// Package postgres stores comments in the PostgreSQL table created by the comment migrations.
package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

const codeInvalidTextRepresentation = "22P02"

// CommentRepo stores comments in the comments table. A comment's article and author are plain IDs: the table
// holds no foreign key to another module's table. Times are read back in UTC.
type CommentRepo struct {
	db database.Querier
}

var _ usecase.CommentRepository = (*CommentRepo)(nil)

// NewCommentRepo returns a CommentRepo that runs its queries on querier. On a pool each call stands alone; on a
// transaction every call commits or rolls back with it.
func NewCommentRepo(querier database.Querier) *CommentRepo {
	return &CommentRepo{db: querier}
}

// translateID returns domain.ErrNotFound for no row and for an ID Postgres refuses as no UUID, and err wrapped
// with operation otherwise.
func translateID(err error, operation string) error {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == codeInvalidTextRepresentation {
		return domain.ErrNotFound
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}
