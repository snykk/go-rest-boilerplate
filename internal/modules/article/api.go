package article

import (
	"context"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

//go:generate go tool mockgen -source=api.go -destination=api_mock_test.go -package=article_test

// ArticleReader is the use case API needs: looking an article up by ID.
type ArticleReader interface {
	// Get returns the article with the given ID and status, of any status when status is empty, or
	// domain.ErrNotFound.
	Get(ctx context.Context, id string, status domain.Status) (usecase.ArticleView, error)
}

// API is what other modules may call on the article module, each through a gateway of its own. It is safe for
// concurrent use.
type API struct {
	articles ArticleReader
}

// NewAPI returns an API over articles.
func NewAPI(articles ArticleReader) *API {
	return &API{articles: articles}
}

// Article returns the article with the given ID, whatever its status. It returns contract.ErrNotFound for an
// unknown ID and the store's error when it fails.
func (a *API) Article(ctx context.Context, id string) (contract.Article, error) {
	view, err := a.articles.Get(ctx, id, "")
	if errors.Is(err, domain.ErrNotFound) {
		return contract.Article{}, contract.ErrNotFound
	}
	if err != nil {
		return contract.Article{}, err
	}
	return contract.Article{
		ID:        view.ID,
		Slug:      view.Slug,
		Title:     view.Title,
		Published: view.Status == domain.StatusPublished,
	}, nil
}
