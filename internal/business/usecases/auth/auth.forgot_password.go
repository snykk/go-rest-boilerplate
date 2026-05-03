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
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// resetTokenBytes is the entropy for the opaque reset token. 32
// bytes (~256 bits) makes brute force on a 30-min window infeasible.
const resetTokenBytes = 32

func resetKey(token string) string { return fmt.Sprintf("pwd_reset:%s", token) }

// ForgotPassword issues a one-shot reset token, persists it in Redis
// with TTL, and emails it to the user. To defeat email enumeration
// the response is identical whether the email exists or not.
func (uc *usecase) ForgotPassword(ctx context.Context, email string) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "ForgotPassword"
		fileName    = "auth.forgot_password.go"
	)
	startTime := time.Now()
	email = domain.NormalizeEmail(email)

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

	user, lookupErr := uc.users.GetByEmail(ctx, email)
	if lookupErr != nil {
		// Swallow NotFound silently to avoid leaking which emails
		// have accounts. Real infra failures still bubble up.
		var domErr *apperror.DomainError
		if errors.As(lookupErr, &domErr) && domErr.Type == apperror.ErrTypeNotFound {
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

	if setErr := uc.redisCache.Set(ctx, resetKey(token), user.ID); setErr != nil {
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
	if expireErr := uc.redisCache.Expire(ctx, resetKey(token), uc.cfg.PasswordResetTTL); expireErr != nil {
		// Non-fatal: token still works, just won't auto-expire.
		logger.ErrorWithContext(ctx, "Forgot password: failed to set TTL on reset token (non-fatal)", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_expire_reset_token",
			"error":   expireErr.Error(),
		})
	}

	if mailErr := uc.mailer.SendPasswordReset(token, email); mailErr != nil {
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
