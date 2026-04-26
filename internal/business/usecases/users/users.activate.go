package users

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

// Activate flips the user's active flag — the only legitimate caller
// is the auth context's VerifyOTP flow. Lives in the User bounded
// context (not Auth) because flipping `active` is an operation on the
// user record, regardless of what triggered it.
func (uc *usecase) Activate(ctx context.Context, userID string) error {
	if err := uc.repo.ChangeActiveUser(ctx, &entities.UserDomain{ID: userID, Active: true}); err != nil {
		return apperror.InternalCause(fmt.Errorf("activate user: %w", err))
	}
	return nil
}
