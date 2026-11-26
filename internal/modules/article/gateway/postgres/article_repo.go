// Package postgres stores articles in the PostgreSQL table created by the article migrations.
package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

const (
	codeUniqueViolation           = "23505"
	codeInvalidTextRepresentation = "22P02"
	slugIndex                     = "articles_slug_key"
)

// ArticleRepo stores articles in the articles table. Times are read back in UTC.
type ArticleRepo struct {
	db database.Querier
}

var _ usecase.ArticleRepository = (*ArticleRepo)(nil)

// NewArticleRepo returns an ArticleRepo that runs its queries on querier. On a pool each call stands alone; on a
// transaction every call commits or rolls back with it, which is how a unit of work groups repositories.
func NewArticleRepo(querier database.Querier) *ArticleRepo {
	return &ArticleRepo{db: querier}
}

func translate(err error, operation string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == codeUniqueViolation && pgErr.ConstraintName == slugIndex {
		return domain.ErrSlugTaken
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func translateID(err error, operation string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == codeInvalidTextRepresentation {
		return domain.ErrNotFound
	}
	return translate(err, operation)
}
