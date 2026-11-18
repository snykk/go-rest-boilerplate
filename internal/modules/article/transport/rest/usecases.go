// Package rest serves the article use cases as a JSON API under /v1/articles, with RFC 9457 problems for
// errors.
package rest

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

//go:generate go tool mockgen -source=usecases.go -destination=usecases_mock_test.go -package=rest_test

// UseCases is what the handlers need from the article use cases; *usecase.ArticleUsecase satisfies it.
type UseCases interface {
	// Create stores a new draft article; errors are the domain errors of usecase.ArticleUsecase.Create.
	Create(ctx context.Context, input usecase.CreateInput) (usecase.ArticleView, error)
	// Get returns the article with the given ID and status, or domain.ErrNotFound.
	Get(ctx context.Context, id string, status domain.Status) (usecase.ArticleView, error)
	// List returns one page of articles.
	List(ctx context.Context, input usecase.ListInput) (usecase.ListOutput, error)
	// Update changes the given fields of an article; errors are the domain errors of usecase.ArticleUsecase.Update.
	Update(ctx context.Context, input usecase.UpdateInput) (usecase.ArticleView, error)
	// Publish publishes a draft; errors are the domain errors of usecase.ArticleUsecase.Publish.
	Publish(ctx context.Context, id string) (usecase.ArticleView, error)
	// Delete removes an article, or returns domain.ErrNotFound.
	Delete(ctx context.Context, id string) error
}

var _ UseCases = (*usecase.ArticleUsecase)(nil)
