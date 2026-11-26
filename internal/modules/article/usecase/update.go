package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// UpdateInput names an article and the fields to change in it; a nil field stays as it is.
type UpdateInput struct {
	ID                string
	Slug, Title, Body *string
}

// Update changes the given fields of the article with ID input.ID and saves it; concurrent updates of one article
// apply one after the other. It returns domain.ErrNotFound for an unknown ID, the errors of
// domain.Article.Edit without saving anything, domain.ErrSlugTaken when another article uses the new slug,
// and the store's error when it fails.
func (u *ArticleUsecase) Update(ctx context.Context, input UpdateInput) (ArticleView, error) {
	article, err := u.articles.Update(ctx, input.ID, func(article *domain.Article) error {
		return article.Edit(input.Slug, input.Title, input.Body, u.clock.Now())
	})
	if err != nil {
		return ArticleView{}, err
	}
	return viewOf(article), nil
}
