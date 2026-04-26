package auth

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
)

// SendOTP generates a 6-digit code, stores it in Redis with TTL, and
// enqueues the email via the async mailer. The HTTP response returns
// on enqueue, not on actual SMTP delivery.
func (uc *usecase) SendOTP(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return apperror.NotFound("email not found")
	}

	if user.Active {
		return apperror.BadRequest("account already activated")
	}

	code, err := helpers.GenerateOTPCode(6)
	if err != nil {
		return apperror.InternalCause(fmt.Errorf("generate otp: %w", err))
	}

	if err = uc.mailer.SendOTP(code, email); err != nil {
		observability.ObserveMailerOp("queue_full")
		logger.Error("failed to enqueue OTP email", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
		return apperror.InternalCause(fmt.Errorf("send otp: %w", err))
	}

	// store OTP code in Redis and reset failed-attempt counter
	otpKey := fmt.Sprintf("user_otp:%s", email)
	if err = uc.redisCache.Set(ctx, otpKey, code); err != nil {
		observability.ObserveCacheOp("redis", "set", "error")
		logger.Error("failed to cache OTP", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	} else {
		observability.ObserveCacheOp("redis", "set", "ok")
	}
	_ = uc.redisCache.Del(ctx, otpAttemptsKey(email))

	return nil
}
