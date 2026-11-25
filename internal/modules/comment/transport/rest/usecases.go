// Package rest serves the comment use cases as a JSON API under /comments, with RFC 9457 problems for errors.
package rest

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

//go:generate go tool mockgen -source=usecases.go -destination=usecases_mock_test.go -package=rest_test

// UseCases is what the handlers need from the comment use cases; *usecase.CommentUsecase satisfies it.
type UseCases interface {
	// Add stores a comment on a published article. Its errors are those of usecase.CommentUsecase.Add:
	// domain.ErrArticleNotFound, domain.ErrAuthorNotFound, domain.ErrInvalidBody or a port's.
	Add(ctx context.Context, input usecase.AddInput) (usecase.CommentView, error)
	// List returns one page of the comments on a published article, or domain.ErrArticleNotFound.
	List(ctx context.Context, input usecase.ListInput) (usecase.ListOutput, error)
	// Delete removes a comment for its author or an editor. Its errors are those of
	// usecase.CommentUsecase.Delete: domain.ErrForbidden, domain.ErrNotFound or the store's.
	Delete(ctx context.Context, input usecase.DeleteInput) error
}

var _ UseCases = (*usecase.CommentUsecase)(nil)
