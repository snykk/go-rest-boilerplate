package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Publish publishes the draft with the given ID at the current time and saves it; a concurrent update of the
// same article applies before or after it, never over it. It returns domain.ErrNotFound for an unknown ID,
// domain.ErrAlreadyPublished or domain.ErrEmptyBody without saving anything, and the store's error when it
// fails.
func (u *ArticleUsecase) Publish(ctx context.Context, id string) (ArticleView, error) {
	article, err := u.articles.Update(ctx, id, func(article *domain.Article) error {
		return article.Publish(u.clock.Now())
	})
	if err != nil {
		return ArticleView{}, err
	}
	return viewOf(article), nil
}
