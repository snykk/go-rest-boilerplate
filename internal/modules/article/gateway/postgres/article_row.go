package postgres

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

const articleColumns = "id, slug, title, body, status, published_at, created_at, updated_at"

type articleRow struct {
	ID          string     `db:"id"`
	Slug        string     `db:"slug"`
	Title       string     `db:"title"`
	Body        string     `db:"body"`
	Status      string     `db:"status"`
	PublishedAt *time.Time `db:"published_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

func (r articleRow) toDomain() *domain.Article {
	stored := domain.Stored{
		ID:        r.ID,
		Slug:      r.Slug,
		Title:     r.Title,
		Body:      r.Body,
		Status:    domain.Status(r.Status),
		CreatedAt: r.CreatedAt.UTC(),
		UpdatedAt: r.UpdatedAt.UTC(),
	}
	if r.PublishedAt != nil {
		publishedAt := r.PublishedAt.UTC()
		stored.PublishedAt = &publishedAt
	}
	return domain.Restore(stored)
}

func rowOf(article *domain.Article) articleRow {
	row := articleRow{
		ID:        article.ID(),
		Slug:      article.Slug(),
		Title:     article.Title(),
		Body:      article.Body(),
		Status:    string(article.Status()),
		CreatedAt: article.CreatedAt(),
		UpdatedAt: article.UpdatedAt(),
	}
	if publishedAt, ok := article.PublishedAt(); ok {
		row.PublishedAt = &publishedAt
	}
	return row
}
