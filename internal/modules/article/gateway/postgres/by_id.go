package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// ByID returns the article with the given ID. It returns domain.ErrNotFound when there is none, including
// when id is not a UUID.
func (r *ArticleRepo) ByID(ctx context.Context, id string) (*domain.Article, error) {
	rows, err := r.db.Query(ctx, `SELECT `+articleColumns+` FROM articles WHERE id = $1`, id)
	if err != nil {
		return nil, translateID(err, "select article")
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[articleRow])
	if err != nil {
		return nil, translateID(err, "select article")
	}
	return row.toDomain(), nil
}
