package postgres

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

// Create inserts comment.
func (r *CommentRepo) Create(ctx context.Context, comment *domain.Comment) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO comments (`+commentColumns+`) VALUES ($1, $2, $3, $4, $5, $6)`,
		comment.ID(), comment.ArticleID(), comment.AuthorID(), comment.AuthorName(), comment.Body(),
		comment.CreatedAt())
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}
