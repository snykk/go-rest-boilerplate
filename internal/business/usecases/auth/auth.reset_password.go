package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// ResetPassword consumes a one-shot reset token (from ForgotPassword)
// and replaces the user's password. The token is deleted on success
// so it can't be replayed.
func (uc *usecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	if newPassword == "" {
		return apperror.BadRequest("new password is required")
	}
	if token == "" {
		return apperror.BadRequest("reset token is required")
	}

	userID, err := uc.redisCache.Get(ctx, resetKey(token))
	if err != nil || userID == "" {
		return apperror.Unauthorized("reset token is invalid or expired")
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.ChangePassword(newPassword, uc.cfg.BcryptCost); err != nil {
		if errors.Is(err, domain.ErrEmptyPassword) {
			return apperror.BadRequest(err.Error())
		}
		return apperror.InternalCause(fmt.Errorf("hash reset password: %w", err))
	}
	if err := uc.users.UpdatePassword(ctx, &user); err != nil {
		return err
	}
	// Best-effort delete; if Redis del fails the token is still
	// invalid because PasswordChangedAt has advanced (Refresh path
	// uses that as the revocation cutoff).
	_ = uc.redisCache.Del(ctx, resetKey(token))
	return nil
}
