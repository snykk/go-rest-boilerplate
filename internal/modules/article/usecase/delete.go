package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
)

// Delete removes the article with the given ID, then announces contract.ArticleDeleted. It returns
// domain.ErrNotFound for an unknown ID and the store's error when it fails; either way it announces nothing.
func (u *ArticleUsecase) Delete(ctx context.Context, id string) error {
	if err := u.articles.Delete(ctx, id); err != nil {
		return err
	}

	u.events.Deleted(ctx, contract.ArticleDeleted{ID: id, At: u.clock.Now()})
	return nil
}
