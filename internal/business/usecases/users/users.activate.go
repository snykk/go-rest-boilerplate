package users

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// Activate flips the user's active flag — the only legitimate caller
// is the auth context's VerifyOTP flow. Lives in the User bounded
// context (not Auth) because flipping `active` is an operation on the
// user record, regardless of what triggered it.
func (uc *usecase) Activate(ctx context.Context, userID string) error {
	u := &domain.User{ID: userID}
	u.Activate() // domain method: sets Active=true and stamps UpdatedAt
	if err := uc.repo.ChangeActiveUser(ctx, u); err != nil {
		return apperror.InternalCause(fmt.Errorf("activate user: %w", err))
	}
	return nil
}
