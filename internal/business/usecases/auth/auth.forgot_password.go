package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// resetTokenBytes is the entropy for the opaque reset token. 32
// bytes (~256 bits) makes brute force on a 30-min window infeasible.
const resetTokenBytes = 32

func resetKey(token string) string { return fmt.Sprintf("pwd_reset:%s", token) }

// ForgotPassword issues a one-shot reset token, persists it in Redis
// with TTL, and emails it to the user. To defeat email enumeration
// the response is identical whether the email exists or not.
func (uc *usecase) ForgotPassword(ctx context.Context, email string) error {
	email = domain.NormalizeEmail(email)
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		// Swallow NotFound silently to avoid leaking which emails
		// have accounts. Real infra failures still bubble up.
		var domErr *apperror.DomainError
		if errors.As(err, &domErr) && domErr.Type == apperror.ErrTypeNotFound {
			return nil
		}
		return err
	}

	token, err := generateResetToken()
	if err != nil {
		return apperror.InternalCause(fmt.Errorf("generate reset token: %w", err))
	}

	if err := uc.redisCache.Set(ctx, resetKey(token), user.ID); err != nil {
		return apperror.InternalCause(fmt.Errorf("persist reset token: %w", err))
	}
	if err := uc.redisCache.Expire(ctx, resetKey(token), uc.cfg.PasswordResetTTL); err != nil {
		// Non-fatal: token still works, just won't auto-expire.
		logger.Error("failed to set TTL on reset token", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"error":                  err.Error(),
		})
	}

	if err := uc.mailer.SendPasswordReset(token, email); err != nil {
		return apperror.InternalCause(fmt.Errorf("send reset email: %w", err))
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
