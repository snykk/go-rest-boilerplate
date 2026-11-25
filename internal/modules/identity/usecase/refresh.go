package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Refresh spends the refresh token and returns a new pair, whose access token carries the roles the user holds
// now, and whose refresh token continues the family of the one it replaces. It returns domain.ErrInvalidToken
// for a token the store refuses, an unknown, revoked, expired or already spent one, and for one whose user
// is gone, and issues no token then. A token the store reports as reused, spent earlier than the grace and its
// family revoked, is refused the same way, after a refresh_reused security event with the user's ID. It
// returns the store's or the signer's error when they fail.
func (u *UserUsecase) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	now := u.clock.Now()
	next, nextToken := u.newSession(now)

	userID, err := u.sessions.Rotate(ctx, hashOf(refreshToken), next, now, u.settings.ReuseGrace)
	if errors.Is(err, domain.ErrTokenReused) {
		u.securityEvent(ctx, slog.LevelWarn, eventRefreshReused, userID)
		return TokenPair{}, domain.ErrInvalidToken
	}
	if err != nil {
		return TokenPair{}, err
	}

	user, err := u.users.ByID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return TokenPair{}, domain.ErrInvalidToken
	}
	if err != nil {
		return TokenPair{}, err
	}

	return u.pairOf(user, nextToken, next.ExpiresAt)
}
