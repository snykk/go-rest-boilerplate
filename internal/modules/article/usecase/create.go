package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// CreateInput is the article Create stores.
type CreateInput struct{ Slug, Title, Body string }

// Create stores a new draft built from input. It returns the errors of domain.New before asking the store,
// domain.ErrSlugTaken when another article uses the slug, and the store's error when it fails.
func (u *ArticleUsecase) Create(ctx context.Context, input CreateInput) (ArticleView, error) {
	article, err := domain.New(u.ids.NewID(), input.Slug, input.Title, input.Body, u.clock.Now())
	if err != nil {
		return ArticleView{}, err
	}

	if err := u.articles.Create(ctx, article); err != nil {
		return ArticleView{}, err
	}
	return viewOf(article), nil
}
