package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

// matching selects the comments on the article $1, by the author $2 unless $2 is NULL.
const matching = `article_id = $1 AND ($2::uuid IS NULL OR author_id = $2)`

// List returns the page of comments query selects, ordered by created_at and then id, newest first, and the
// number of all comments on query.ArticleID, by query.AuthorID unless it is empty.
func (r *CommentRepo) List(ctx context.Context, query usecase.CommentQuery) ([]domain.Comment, int, error) {
	var authorID *string
	if query.AuthorID != "" {
		authorID = &query.AuthorID
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+commentColumns+` FROM comments
		  WHERE `+matching+`
		  ORDER BY created_at DESC, id DESC
		  LIMIT $3 OFFSET $4`,
		query.ArticleID, authorID, query.Limit, query.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list comments: %w", err)
	}
	found, err := pgx.CollectRows(rows, pgx.RowToStructByName[commentRow])
	if err != nil {
		return nil, 0, fmt.Errorf("list comments: %w", err)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT count(*) FROM comments WHERE `+matching, query.ArticleID, authorID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	items := make([]domain.Comment, len(found))
	for i, row := range found {
		items[i] = *row.toDomain()
	}
	return items, total, nil
}
