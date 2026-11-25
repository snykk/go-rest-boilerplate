package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

// ByID returns the comment with the given ID. It returns domain.ErrNotFound when there is none, including when
// id is not a UUID.
func (r *CommentRepo) ByID(ctx context.Context, id string) (*domain.Comment, error) {
	rows, err := r.db.Query(ctx, `SELECT `+commentColumns+` FROM comments WHERE id = $1`, id)
	if err != nil {
		return nil, translateID(err, "select comment")
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[commentRow])
	if err != nil {
		return nil, translateID(err, "select comment")
	}
	return row.toDomain(), nil
}
