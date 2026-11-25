package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

// AddInput is the comment Add stores: Body by the user with AuthorID on the article with ArticleID.
type AddInput struct{ ArticleID, AuthorID, Body string }

// Add stores a comment built from input on a published article, with the display name its author has now. It
// returns domain.ErrArticleNotFound when the article does not exist or is a draft, domain.ErrAuthorNotFound when
// no user has the author's ID, domain.ErrInvalidBody for a body the domain refuses, before asking the store,
// and the error of a port that fails.
func (u *CommentUsecase) Add(ctx context.Context, input AddInput) (CommentView, error) {
	if err := u.requirePublished(ctx, input.ArticleID); err != nil {
		return CommentView{}, err
	}
	authorName, err := u.authors.DisplayName(ctx, input.AuthorID)
	if err != nil {
		return CommentView{}, err
	}

	comment, err := domain.New(u.ids.NewID(), input.ArticleID, input.AuthorID, authorName, input.Body, u.clock.Now())
	if err != nil {
		return CommentView{}, err
	}
	if err := u.comments.Create(ctx, comment); err != nil {
		return CommentView{}, err
	}
	return viewOf(comment), nil
}
