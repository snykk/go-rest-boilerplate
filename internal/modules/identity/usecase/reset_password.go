package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

const resetAttempts = 1

// ResetPassword spends the password reset token, sets the password of its user and ends any lockout of the
// account, and revokes every session of that user, in one unit of work. It returns domain.ErrWeakPassword
// before asking the store, and domain.ErrInvalidToken for a token that is unknown, expired or already used; an
// unknown token is refused before the password is hashed, so a made-up token costs no hash. It returns the
// store's or the hasher's error when they fail. Once the unit of work commits, it logs a password_reset
// security event with the user's ID.
func (u *UserUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	plain, err := domain.NewPassword(newPassword)
	if err != nil {
		return err
	}

	tokenHash := hashOf(token)
	if _, err := u.resets.ByHash(ctx, tokenHash); err != nil {
		return err
	}

	hash, err := u.hasher.Hash(plain.Reveal())
	if err != nil {
		return err
	}

	now := u.clock.Now()
	var userID string
	err = u.unitOfWork.Do(ctx, func(repositories Repositories) error {
		var err error
		userID, err = spendResetToken(ctx, repositories.Resets, tokenHash, now)
		if err != nil {
			return err
		}

		if _, err := repositories.Users.Update(ctx, userID, resetPassword(hash, now)); err != nil {
			return err
		}
		return repositories.Sessions.RevokeAll(ctx, userID, now)
	})
	if err != nil {
		return err
	}

	u.securityEvent(ctx, slog.LevelInfo, eventPasswordReset, userID)
	return nil
}

func resetPassword(hash string, now time.Time) func(user *domain.User) error {
	return func(user *domain.User) error {
		user.ResetPassword(hash, now)
		return nil
	}
}

// spendResetToken uses up the reset token with the given hash and returns the ID of its user. Whatever
// makes the token unusable, it returns domain.ErrInvalidToken.
func spendResetToken(ctx context.Context, resets SecretRepository, tokenHash string, now time.Time) (string, error) {
	stored, err := resets.ByHash(ctx, tokenHash)
	if err != nil {
		return "", err
	}

	err = resets.Use(ctx, stored.UserID, func(secret *domain.Secret) error {
		return secret.Check(tokenHash, now, resetAttempts)
	})
	switch {
	case errors.Is(err, domain.ErrInvalidCode),
		errors.Is(err, domain.ErrCodeExpired),
		errors.Is(err, domain.ErrTooManyAttempts):
		return "", domain.ErrInvalidToken
	case err != nil:
		return "", err
	}
	return stored.UserID, nil
}
