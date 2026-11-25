package usecase

import (
	"context"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Verify marks verified the email of the user with the given email, when code is the code they hold. It
// returns domain.ErrInvalidCode for a wrong code, for an email that is invalid or that no user has, and for a
// user who holds no code, domain.ErrCodeExpired for an expired one and domain.ErrTooManyAttempts when the
// code has been guessed at too often; such a code stays refused, even when it is right, until a new one
// replaces it. A code is spent by the call that verifies, and a wrong guess counts against it. It returns the
// store's error when it fails; when that happens after the code was spent, the user asks for a new one.
func (u *UserUsecase) Verify(ctx context.Context, email, code string) error {
	user, found, err := u.lookup(ctx, email)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrInvalidCode
	}

	now, hash := u.clock.Now(), hashOf(code)
	err = u.verifications.Use(ctx, user.ID(), func(secret *domain.Secret) error {
		return secret.Check(hash, now, u.settings.CodeAttempts)
	})
	if err != nil {
		return err
	}

	return u.markVerified(ctx, user.ID(), now)
}

func (u *UserUsecase) markVerified(ctx context.Context, userID string, now time.Time) error {
	_, err := u.users.Update(ctx, userID, func(user *domain.User) error {
		user.Verify(now)
		return nil
	})
	return err
}
