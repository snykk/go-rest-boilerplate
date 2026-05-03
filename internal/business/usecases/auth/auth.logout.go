package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Logout revokes the refresh token by deleting its jti from Redis.
// Access tokens remain valid until their natural expiry — clients
// should discard them on logout. (A full access-token blacklist
// would cost a Redis hop per request and is deliberately out of
// scope for this boilerplate.)
func (uc *usecase) Logout(ctx context.Context, req LogoutRequest) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "Logout"
		fileName    = "auth.logout.go"
	)
	startTime := time.Now()
	refreshToken := req.RefreshToken

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"has_refresh_token": refreshToken != "",
		},
	})

	defer func() {
		duration := time.Since(startTime)
		fields := logger.Fields{
			"usecase":  usecaseName,
			"method":   funcName,
			"file":     fileName,
			"duration": duration.Milliseconds(),
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	claims, parseErr := uc.jwtService.ParseRefreshToken(refreshToken)
	if parseErr != nil {
		err = apperror.Unauthorized("invalid refresh token")
		logger.ErrorWithContext(ctx, "Logout failed: invalid token", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "parse_refresh_token",
			"error":   parseErr.Error(),
		})
		return err
	}
	if delErr := uc.redisCache.Del(ctx, refreshKey(claims.ID)); delErr != nil {
		err = apperror.InternalCause(fmt.Errorf("revoke refresh: %w", delErr))
		logger.ErrorWithContext(ctx, "Logout failed: redis del error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_del",
			"error":   delErr.Error(),
			"jti":     claims.ID,
		})
		return err
	}
	return nil
}
