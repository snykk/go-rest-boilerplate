package auth

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
)

// Logout revokes the refresh token by deleting its jti from Redis.
// Access tokens remain valid until their natural expiry — clients
// should discard them on logout. (A full access-token blacklist
// would cost a Redis hop per request and is deliberately out of
// scope for this boilerplate.)
func (uc *usecase) Logout(ctx context.Context, refreshToken string) error {
	claims, err := uc.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		return apperror.Unauthorized("invalid refresh token")
	}
	if err := uc.redisCache.Del(ctx, refreshKey(claims.ID)); err != nil {
		return apperror.InternalCause(fmt.Errorf("revoke refresh: %w", err))
	}
	return nil
}
