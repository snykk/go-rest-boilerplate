package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// ChangePassword swaps the authenticated user's password after
// verifying the current one. The new PasswordChangedAt timestamp acts
// as a revocation cutoff — refresh tokens issued before it are
// rejected on /refresh.
func (uc *usecase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if newPassword == "" {
		return apperror.BadRequest("new password is required")
	}
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.VerifyPassword(currentPassword) {
		return apperror.Unauthorized("current password is incorrect")
	}
	if err := user.ChangePassword(newPassword, uc.cfg.BcryptCost); err != nil {
		if errors.Is(err, domain.ErrEmptyPassword) {
			return apperror.BadRequest(err.Error())
		}
		return apperror.InternalCause(fmt.Errorf("hash new password: %w", err))
	}
	return uc.users.UpdatePassword(ctx, &user)
}
