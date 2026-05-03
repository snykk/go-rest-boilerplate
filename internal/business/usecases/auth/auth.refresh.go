package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Refresh verifies the supplied refresh token, mints a new
// access+refresh pair, and revokes the old jti in Redis. Replay of
// an already-used refresh token fails because rememberRefresh → Del
// makes the old jti unknown.
func (uc *usecase) Refresh(ctx context.Context, refreshToken string) (out LoginResult, err error) {
	const (
		usecaseName = "auth"
		funcName    = "Refresh"
		fileName    = "auth.refresh.go"
	)
	startTime := time.Now()

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
		if err == nil {
			fields["response"] = logger.Fields{"user_id": out.User.ID}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	claims, parseErr := uc.jwtService.ParseRefreshToken(refreshToken)
	if parseErr != nil {
		err = apperror.Unauthorized("invalid refresh token")
		logger.ErrorWithContext(ctx, "Refresh failed: invalid token", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "parse_refresh_token",
			"error":   parseErr.Error(),
		})
		return LoginResult{}, err
	}

	// Verify the jti is still live server-side; logout / previous
	// rotation would have removed it.
	if _, getErr := uc.redisCache.Get(ctx, refreshKey(claims.ID)); getErr != nil {
		err = apperror.Unauthorized("refresh token has been revoked")
		logger.ErrorWithContext(ctx, "Refresh failed: token revoked", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "check_jti_alive",
			"error":   getErr.Error(),
			"jti":     claims.ID,
		})
		return LoginResult{}, err
	}

	// Fresh identity lookup so revoked / deactivated accounts stop
	// getting new access tokens even while their refresh is live.
	user, lookupErr := uc.users.GetByEmail(ctx, claims.Email)
	if lookupErr != nil {
		err = apperror.Unauthorized("user no longer exists")
		logger.ErrorWithContext(ctx, "Refresh failed: user lookup error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "get_user_by_email",
			"error":   lookupErr.Error(),
			"email":   claims.Email,
		})
		return LoginResult{}, err
	}
	if !user.Active {
		err = apperror.Forbidden("account is not activated")
		logger.ErrorWithContext(ctx, "Refresh failed: account not activated", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "check_active",
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	// Reject tokens issued before the most recent password change —
	// rotating the password must close pre-existing sessions.
	if cutoff := user.TokensRevokedBefore(); !cutoff.IsZero() &&
		claims.IssuedAt != nil && claims.IssuedAt.Time.Before(cutoff) {
		err = apperror.Unauthorized("refresh token has been revoked")
		logger.ErrorWithContext(ctx, "Refresh failed: token issued before password rotation", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "check_revocation_cutoff",
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	pair, mintErr := uc.jwtService.GenerateTokenPair(user.ID, user.IsAdmin(), user.Email)
	if mintErr != nil {
		err = apperror.InternalCause(fmt.Errorf("generate token: %w", mintErr))
		logger.ErrorWithContext(ctx, "Refresh failed: token generation error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "generate_token_pair",
			"error":   mintErr.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	// Rotate: remove the old jti, record the new one. Do this after
	// the new pair is minted so a mint failure doesn't leave the
	// user with no valid refresh token at all.
	if persistErr := uc.rememberRefresh(ctx, pair); persistErr != nil {
		err = apperror.InternalCause(fmt.Errorf("persist refresh: %w", persistErr))
		logger.ErrorWithContext(ctx, "Refresh failed: persist refresh error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "persist_refresh",
			"error":   persistErr.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}
	_ = uc.redisCache.Del(ctx, refreshKey(claims.ID))

	out = LoginResult{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}
	return out, nil
}
