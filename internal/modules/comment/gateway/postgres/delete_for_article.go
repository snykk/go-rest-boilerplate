package postgres

import (
	"context"
	"fmt"
)

// DeleteForArticle removes every comment on the article with the given ID. It returns nil when the article has
// none, so calling it again changes nothing.
func (r *CommentRepo) DeleteForArticle(ctx context.Context, articleID string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM comments WHERE article_id = $1`, articleID); err != nil {
		return fmt.Errorf("delete comments of article: %w", err)
	}
	return nil
}
