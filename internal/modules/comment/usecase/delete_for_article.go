package usecase

import "context"

// DeleteForArticle removes every comment on the article with the given ID, whatever the article's state. An
// article with no comments is not an error, so calling it again changes nothing. It returns the store's error
// when it fails.
func (u *CommentUsecase) DeleteForArticle(ctx context.Context, articleID string) error {
	return u.comments.DeleteForArticle(ctx, articleID)
}
