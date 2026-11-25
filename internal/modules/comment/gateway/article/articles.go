// Package article tells the comment module whether an article takes comments, by asking the article module's
// API: an anti-corruption layer that keeps that module's model out of this one.
package article

import (
	"context"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

//go:generate go tool mockgen -source=articles.go -destination=articles_mock_test.go -package=article_test

// API is the part of the article module's API that Articles asks.
type API interface {
	// Article returns the article with the given ID, whatever its status, or contract.ErrNotFound.
	Article(ctx context.Context, id string) (contract.Article, error)
}

// Articles implements usecase.Articles over the article module's API. It is safe for concurrent use when its
// API is.
type Articles struct {
	api API
}

var _ usecase.Articles = (*Articles)(nil)

// NewArticles returns Articles that ask api.
func NewArticles(api API) *Articles {
	return &Articles{api: api}
}

// Published reports whether the article with the given ID is published. It returns false, and no error, for a
// draft and for an article the API reports as contract.ErrNotFound, and the API's other errors as they are.
func (a *Articles) Published(ctx context.Context, id string) (bool, error) {
	found, err := a.api.Article(ctx, id)
	if errors.Is(err, contract.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found.Published, nil
}
