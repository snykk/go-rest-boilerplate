package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

// Delete removes the comment with the given ID. It returns domain.ErrNotFound when there is none, including when
// id is not a UUID.
func (r *CommentRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM comments WHERE id = $1`, id)
	if err != nil {
		return translateID(err, "delete comment")
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
