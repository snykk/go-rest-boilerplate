package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Login signs in the user with the given email and password, and starts a new family of sessions for them.
// It returns domain.ErrInvalidCredentials at once, counting nothing, for an email the domain refuses and for a
// password outside 8 to 72 bytes, which no user has. For an email no user has, it returns the same error after
// comparing the password with a dummy hash, so the time it takes does not tell it from a wrong password; such
// an attempt counts against no account. A wrong password returns domain.ErrInvalidCredentials and counts as a
// failed sign-in of the user.
//
// A locked account returns domain.ErrAccountLocked before the password is compared. The lock is checked again
// under the user's row lock when the attempt is saved, before any other answer: an attempt that read the account
// before a concurrent one locked it returns domain.ErrAccountLocked too, with a wrong password or the right one,
// so once the account is locked no answer tells the right password apart. Then a right password returns
// domain.ErrEmailNotVerified for an unverified email, and otherwise clears the count of failed sign-ins. It
// returns the store's, the hasher's or the signer's error when they fail.
//
// It logs security events: login_succeeded once the session is stored, login_failed for every wrong password
// and every domain.ErrInvalidCredentials, and account_locked when the failure is the one that locks the
// account, once per lock. login_failed carries the user's ID, except when the email is invalid, the password is
// outside 8 to 72 bytes or no user has the email: those are refused before a user is known.
func (u *UserUsecase) Login(ctx context.Context, email, password string) (TokenPair, error) {
	address, err := domain.NewEmail(email)
	if err != nil {
		return TokenPair{}, u.failWithoutUser(ctx)
	}
	plain, err := domain.NewPassword(password)
	if err != nil {
		return TokenPair{}, u.failWithoutUser(ctx)
	}

	user, err := u.users.ByEmail(ctx, address)
	if errors.Is(err, domain.ErrNotFound) {
		u.hasher.CompareDummy(plain.Reveal())
		return TokenPair{}, u.failWithoutUser(ctx)
	}
	if err != nil {
		return TokenPair{}, err
	}

	now := u.clock.Now()
	if user.Locked(now) {
		return TokenPair{}, domain.ErrAccountLocked
	}

	if !u.hasher.Compare(user.PasswordHash(), plain.Reveal()) {
		return TokenPair{}, u.recordFailure(ctx, user.ID(), now)
	}
	if err := u.confirmLogin(ctx, user.ID(), now); err != nil {
		return TokenPair{}, err
	}

	session, refreshToken := u.newFamily(user, now)
	pair, err := u.pairOf(user, refreshToken, session.ExpiresAt)
	if err != nil {
		return TokenPair{}, err
	}

	if err := u.sessions.Create(ctx, session); err != nil {
		return TokenPair{}, err
	}

	u.securityEvent(ctx, slog.LevelInfo, eventLoginSucceeded, user.ID())
	return pair, nil
}

// failWithoutUser logs a failed sign-in of no known user and returns domain.ErrInvalidCredentials.
func (u *UserUsecase) failWithoutUser(ctx context.Context) error {
	u.securityEvent(ctx, slog.LevelWarn, eventLoginFailed, "")
	return domain.ErrInvalidCredentials
}

// confirmLogin checks, under the user's row lock, that the account is not locked, which a concurrent attempt
// may have done since it was read, and that the email is verified, then clears the count of failed sign-ins. It
// returns domain.ErrAccountLocked, then domain.ErrEmailNotVerified, saving nothing.
func (u *UserUsecase) confirmLogin(ctx context.Context, userID string, now time.Time) error {
	_, err := u.users.Update(ctx, userID, func(user *domain.User) error {
		if user.Locked(now) {
			return domain.ErrAccountLocked
		}
		if !user.Verified() {
			return domain.ErrEmailNotVerified
		}
		user.SucceedLogin()
		return nil
	})
	return err
}
