package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// resetTokenBytes is the entropy for the opaque reset token. 32
// bytes (~256 bits) makes brute force on a 30-min window infeasible.
const resetTokenBytes = 32

// ForgotPassword issues a one-shot reset token, persists it in Redis
// with TTL, and emails it to the user. To defeat email enumeration
// the response is identical whether the email exists or not.
func (uc *usecase) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "ForgotPassword"
		fileName    = "auth.forgot_password.go"
	)
	startTime := time.Now()
	email := domain.NormalizeEmail(req.Email)

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"email": email,
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

	if uc.cfg.ForgotMaxAttempts > 0 {
		key := ForgotAttemptsKey(email)
		attempts, incrErr := uc.redisCache.Incr(ctx, key)
		if incrErr != nil {
			logger.ErrorWithContext(ctx, "ForgotPassword: failed to track attempts (non-fatal)", logger.Fields{
				"usecase": usecaseName,
				"method":  funcName,
				"file":    fileName,
				"step":    "redis_incr_attempts",
				"error":   incrErr.Error(),
				"email":   email,
			})
		} else if attempts == 1 {
			_ = uc.redisCache.Expire(ctx, key, uc.cfg.ForgotLockoutTTL)
		}
		if attempts > int64(uc.cfg.ForgotMaxAttempts) {
			err = apperror.Forbidden("too many password reset requests, please try again later")
			logger.ErrorWithContext(ctx, "ForgotPassword failed: rate limit exceeded", logger.Fields{
				"usecase":  usecaseName,
				"method":   funcName,
				"file":     fileName,
				"step":     "check_rate_limit",
				"error":    err.Error(),
				"email":    email,
				"attempts": attempts,
			})
			return err
		}
	}

	// Generate a token unconditionally so the unknown-email path does
	// the same crypto work as the known-email path. Defeats timing-based
	// email enumeration, complementing the identical 200-OK response.
	token, tokenErr := generateResetToken()
	if tokenErr != nil {
		err = apperror.InternalCause(fmt.Errorf("generate reset token: %w", tokenErr))
		logger.ErrorWithContext(ctx, "Forgot password failed: token generation error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "generate_reset_token",
			"error":   tokenErr.Error(),
			"email":   email,
		})
		return err
	}

	lookupResp, lookupErr := uc.users.GetByEmail(ctx, users.GetByEmailRequest{Email: email})
	if lookupErr != nil {
		var domErr *apperror.DomainError
		if errors.As(lookupErr, &domErr) && domErr.Type == apperror.ErrTypeNotFound {
			// Hedge: do a Redis write of equivalent shape so unknown
			// emails take roughly the same time as known ones.
			decoyKey := PasswordResetKey(token)
			_ = uc.redisCache.Set(ctx, decoyKey, "decoy")
			_ = uc.redisCache.Expire(ctx, decoyKey, uc.cfg.PasswordResetTTL)
			_ = uc.redisCache.Del(ctx, decoyKey)
			return nil
		}
		err = lookupErr
		logger.ErrorWithContext(ctx, "Forgot password failed: user lookup error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "get_user_by_email",
			"error":   lookupErr.Error(),
			"email":   email,
		})
		return err
	}
	user := lookupResp.User

	// Invalidate any token still live for this user so a leaked earlier
	// link can't race with the new one. Single-active-token is the
	// expected mental model for "I requested a reset twice".
	if prior, getErr := uc.redisCache.Get(ctx, UserResetIndexKey(user.ID)); getErr == nil && prior != "" {
		_ = uc.redisCache.Del(ctx, PasswordResetKey(prior))
	}

	if setErr := uc.redisCache.Set(ctx, PasswordResetKey(token), user.ID); setErr != nil {
		err = apperror.InternalCause(fmt.Errorf("persist reset token: %w", setErr))
		logger.ErrorWithContext(ctx, "Forgot password failed: persist token error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_set_reset_token",
			"error":   setErr.Error(),
			"user_id": user.ID,
		})
		return err
	}
	if expireErr := uc.redisCache.Expire(ctx, PasswordResetKey(token), uc.cfg.PasswordResetTTL); expireErr != nil {
		logger.ErrorWithContext(ctx, "Forgot password: failed to set TTL on reset token (non-fatal)", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_expire_reset_token",
			"error":   expireErr.Error(),
		})
	}
	if setIdxErr := uc.redisCache.Set(ctx, UserResetIndexKey(user.ID), token); setIdxErr != nil {
		logger.ErrorWithContext(ctx, "Forgot password: failed to update user reset index (non-fatal)", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_set_user_index",
			"error":   setIdxErr.Error(),
			"user_id": user.ID,
		})
	} else {
		_ = uc.redisCache.Expire(ctx, UserResetIndexKey(user.ID), uc.cfg.PasswordResetTTL)
	}

	if mailErr := uc.mailer.SendPasswordReset(ctx, token, email); mailErr != nil {
		err = apperror.InternalCause(fmt.Errorf("send reset email: %w", mailErr))
		logger.ErrorWithContext(ctx, "Forgot password failed: mailer error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "mailer_send_password_reset",
			"error":   mailErr.Error(),
			"email":   email,
		})
		return err
	}
	return nil
}

func generateResetToken() (string, error) {
	buf := make([]byte, resetTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
