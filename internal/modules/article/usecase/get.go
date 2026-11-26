package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Get returns the article with the given ID when it has the given status, or with any status when status is
// empty. It returns domain.ErrNotFound for an unknown ID or an article of another status, and the store's
// error when it fails.
func (u *ArticleUsecase) Get(ctx context.Context, id string, status domain.Status) (ArticleView, error) {
	article, err := u.articles.ByID(ctx, id)
	if err != nil {
		return ArticleView{}, err
	}
	if status != "" && article.Status() != status {
		return ArticleView{}, domain.ErrNotFound
	}
	return viewOf(article), nil
}
