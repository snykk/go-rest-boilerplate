package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// List returns the page of articles query selects, ordered by created_at and then id, newest first, and the
// number of all articles matching query.Status.
func (r *ArticleRepo) List(ctx context.Context, query usecase.ArticleQuery) ([]domain.Article, int, error) {
	status := string(query.Status)
	rows, err := r.db.Query(ctx,
		`SELECT `+articleColumns+` FROM articles
		  WHERE ($1::text = '' OR status = $1)
		  ORDER BY created_at DESC, id DESC
		  LIMIT $2 OFFSET $3`,
		status, query.Limit, query.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list articles: %w", err)
	}
	found, err := pgx.CollectRows(rows, pgx.RowToStructByName[articleRow])
	if err != nil {
		return nil, 0, fmt.Errorf("list articles: %w", err)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT count(*) FROM articles WHERE ($1::text = '' OR status = $1)`, status).
		Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}

	items := make([]domain.Article, len(found))
	for i, row := range found {
		items[i] = *row.toDomain()
	}
	return items, total, nil
}
