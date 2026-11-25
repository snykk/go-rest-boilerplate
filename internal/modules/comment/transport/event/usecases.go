// Package event runs the comment use cases when another module announces an event, as the rest package does
// when a request arrives.
package event

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

//go:generate go tool mockgen -source=usecases.go -destination=usecases_mock_test.go -package=event_test

// UseCases is what the event handlers need from the comment use cases; *usecase.CommentUsecase satisfies it.
type UseCases interface {
	// DeleteForArticle removes every comment on the article with the given ID. Calling it again changes nothing.
	DeleteForArticle(ctx context.Context, articleID string) error
}

var _ UseCases = (*usecase.CommentUsecase)(nil)
