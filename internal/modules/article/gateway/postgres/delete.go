package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Delete removes the article with the given ID. It returns domain.ErrNotFound when there is none, including
// when id is not a UUID.
func (r *ArticleRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM articles WHERE id = $1`, id)
	if err != nil {
		return translateID(err, "delete article")
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
