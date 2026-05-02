package auth

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
)

// Refresh verifies the supplied refresh token, mints a new
// access+refresh pair, and revokes the old jti in Redis. Replay of
// an already-used refresh token fails because rememberRefresh → Del
// makes the old jti unknown.
func (uc *usecase) Refresh(ctx context.Context, refreshToken string) (LoginResult, error) {
	claims, err := uc.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		return LoginResult{}, apperror.Unauthorized("invalid refresh token")
	}

	// Verify the jti is still live server-side; logout / previous
	// rotation would have removed it.
	if _, err := uc.redisCache.Get(ctx, refreshKey(claims.ID)); err != nil {
		return LoginResult{}, apperror.Unauthorized("refresh token has been revoked")
	}

	// Fresh identity lookup so revoked / deactivated accounts stop
	// getting new access tokens even while their refresh is live.
	user, err := uc.users.GetByEmail(ctx, claims.Email)
	if err != nil {
		return LoginResult{}, apperror.Unauthorized("user no longer exists")
	}
	if !user.Active {
		return LoginResult{}, apperror.Forbidden("account is not activated")
	}

	// Reject tokens issued before the most recent password change —
	// rotating the password must close pre-existing sessions.
	if cutoff := user.TokensRevokedBefore(); !cutoff.IsZero() &&
		claims.IssuedAt != nil && claims.IssuedAt.Time.Before(cutoff) {
		return LoginResult{}, apperror.Unauthorized("refresh token has been revoked")
	}

	pair, err := uc.jwtService.GenerateTokenPair(user.ID, user.IsAdmin(), user.Email)
	if err != nil {
		return LoginResult{}, apperror.InternalCause(fmt.Errorf("generate token: %w", err))
	}

	// Rotate: remove the old jti, record the new one. Do this after
	// the new pair is minted so a mint failure doesn't leave the
	// user with no valid refresh token at all.
	if err := uc.rememberRefresh(ctx, pair); err != nil {
		return LoginResult{}, apperror.InternalCause(fmt.Errorf("persist refresh: %w", err))
	}
	_ = uc.redisCache.Del(ctx, refreshKey(claims.ID))

	return LoginResult{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}
