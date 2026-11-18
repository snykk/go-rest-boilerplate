// Package usecase runs the article use cases and declares the ports they need from the outside world.
package usecase

import (
	"context"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

//go:generate go tool mockgen -source=ports.go -destination=ports_mock_test.go -package=usecase_test

// ArticleRepository stores and loads articles.
type ArticleRepository interface {
	// Create stores a new article. It returns domain.ErrSlugTaken when the slug is already used.
	Create(ctx context.Context, article *domain.Article) error
	// Update loads the article with the given ID, applies change to it and saves every field, as one step that
	// concurrent updates of the same article wait for, and returns the saved article. It returns
	// domain.ErrNotFound for an unknown ID, change's error without saving anything, and domain.ErrSlugTaken
	// when the new slug is already used.
	Update(ctx context.Context, id string, change func(article *domain.Article) error) (*domain.Article, error)
	// ByID returns the article with the given ID, or domain.ErrNotFound.
	ByID(ctx context.Context, id string) (*domain.Article, error)
	// Delete removes the article with the given ID. It returns domain.ErrNotFound for an unknown ID.
	Delete(ctx context.Context, id string) error
	// List returns one page of the articles matching query, newest first, and the number of all matches.
	List(ctx context.Context, query ArticleQuery) (items []domain.Article, total int, err error)
}

// Clock provides the current time.
type Clock interface {
	// Now returns the current time in UTC, at the precision the article store keeps, so a stored time
	// reads back equal.
	Now() time.Time
}

// IDGenerator provides identifiers for new articles.
type IDGenerator interface {
	// NewID returns an identifier no other call returns, in the form ArticleRepository accepts as an ID.
	NewID() string
}

// ArticleQuery narrows ArticleRepository.List. An empty Status matches every status.
type ArticleQuery struct {
	Status domain.Status
	Limit  int
	Offset int
}
