package usecase

import "context"

// Delete removes the article with the given ID. It returns domain.ErrNotFound for an unknown ID and the
// store's error when it fails.
func (u *ArticleUsecase) Delete(ctx context.Context, id string) error {
	return u.articles.Delete(ctx, id)
}
