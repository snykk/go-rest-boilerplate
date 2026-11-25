package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ChangePasswordInput is the password change ChangePassword makes: the user, the password they have now and
// the one they want.
type ChangePasswordInput struct{ UserID, Current, New string }

// ChangePassword sets the password of the user to the new one when they know the current one. In one unit of
// work it saves the new hash, revokes every session of the user and starts a new family, and it returns the
// pair of that family's first session. It returns domain.ErrWeakPassword for a new password the domain
// refuses, before asking the store, domain.ErrNotFound for an unknown user, and domain.ErrAccountLocked for a
// locked one: before the current password is compared, then under the user's row lock right after it, before
// a wrong current password is counted or a right one leads to hashing the new password, and a last time when
// the change is saved. So once a concurrent attempt has locked the account, a wrong current password and the
// right one take the same path to that answer, one compare and one row-locked update, and neither hashes the
// new password. A wrong current password counts as a failed sign-in and otherwise returns
// domain.ErrInvalidCredentials, as does one outside 8 to 72 bytes, which counts nothing; the password is not
// changed then. It returns the store's, the hasher's or the signer's error when they fail.
//
// It logs security events: password_changed once the unit of work commits, and for a wrong current password
// login_failed, and account_locked when the failure is the one that locks the account, once per lock.
func (u *UserUsecase) ChangePassword(ctx context.Context, input ChangePasswordInput) (TokenPair, error) {
	replacement, err := domain.NewPassword(input.New)
	if err != nil {
		return TokenPair{}, err
	}

	user, err := u.users.ByID(ctx, input.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	now := u.clock.Now()
	if user.Locked(now) {
		return TokenPair{}, domain.ErrAccountLocked
	}

	current, err := domain.NewPassword(input.Current)
	if err != nil {
		u.securityEvent(ctx, slog.LevelWarn, eventLoginFailed, user.ID())
		return TokenPair{}, domain.ErrInvalidCredentials
	}
	if !u.hasher.Compare(user.PasswordHash(), current.Reveal()) {
		return TokenPair{}, u.recordFailure(ctx, user.ID(), now)
	}
	if _, err := u.users.Update(ctx, user.ID(), refuseLocked(now)); err != nil {
		return TokenPair{}, err
	}

	hash, err := u.hasher.Hash(replacement.Reveal())
	if err != nil {
		return TokenPair{}, err
	}

	session, refreshToken := u.newFamily(user, now)
	pair, err := u.pairOf(user, refreshToken, session.ExpiresAt)
	if err != nil {
		return TokenPair{}, err
	}

	err = u.unitOfWork.Do(ctx, func(repositories Repositories) error {
		if _, err := repositories.Users.Update(ctx, user.ID(), setPasswordUnlessLocked(hash, now)); err != nil {
			return err
		}
		if err := repositories.Sessions.RevokeAll(ctx, user.ID(), now); err != nil {
			return err
		}
		return repositories.Sessions.Create(ctx, session)
	})
	if err != nil {
		return TokenPair{}, err
	}

	u.securityEvent(ctx, slog.LevelInfo, eventPasswordChanged, user.ID())
	return pair, nil
}

// refuseLocked returns domain.ErrAccountLocked, so nothing is saved, for a user locked at now, and leaves any
// other user as it is.
func refuseLocked(now time.Time) func(user *domain.User) error {
	return func(user *domain.User) error {
		if user.Locked(now) {
			return domain.ErrAccountLocked
		}
		return nil
	}
}

func setPasswordUnlessLocked(hash string, now time.Time) func(user *domain.User) error {
	return func(user *domain.User) error {
		if user.Locked(now) {
			return domain.ErrAccountLocked
		}
		user.SetPasswordHash(hash, now)
		return nil
	}
}
