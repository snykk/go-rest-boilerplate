package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Create inserts article. It returns domain.ErrSlugTaken when another article uses the slug.
func (r *ArticleRepo) Create(ctx context.Context, article *domain.Article) error {
	row := rowOf(article)
	_, err := r.db.Exec(ctx,
		`INSERT INTO articles (`+articleColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		row.ID, row.Slug, row.Title, row.Body, row.Status, row.PublishedAt, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return translate(err, "insert article")
	}
	return nil
}
