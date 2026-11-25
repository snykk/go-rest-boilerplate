package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

// DeleteInput names the comment Delete removes, ID, and who asks: the user with CallerID, an editor when
// CallerIsEditor is true.
type DeleteInput struct {
	ID             string
	CallerID       string
	CallerIsEditor bool
}

// Delete removes the comment input names when the caller wrote it or is an editor. It returns
// domain.ErrForbidden, removing nothing, for any other caller, domain.ErrNotFound for an unknown comment, and the
// store's error when it fails.
func (u *CommentUsecase) Delete(ctx context.Context, input DeleteInput) error {
	comment, err := u.comments.ByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if comment.AuthorID() != input.CallerID && !input.CallerIsEditor {
		return domain.ErrForbidden
	}

	return u.comments.Delete(ctx, input.ID)
}
