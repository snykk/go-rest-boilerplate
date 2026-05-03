package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
)

// SendOTP generates a 6-digit code, stores it in Redis with TTL, and
// enqueues the email via the async mailer. The HTTP response returns
// on enqueue, not on actual SMTP delivery.
func (uc *usecase) SendOTP(ctx context.Context, email string) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "SendOTP"
		fileName    = "auth.send_otp.go"
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
		err = apperror.NotFound("email not found")
		logger.ErrorWithContext(ctx, "Send OTP failed: user lookup error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "get_user_by_email",
			"error":   lookupErr.Error(),
			"email":   email,
		})
		return err
	}

	if user.Active {
		err = apperror.BadRequest("account already activated")
		logger.ErrorWithContext(ctx, "Send OTP failed: account already activated", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "check_active",
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return err
	}

	code, otpErr := helpers.GenerateOTPCode(6)
	if otpErr != nil {
		err = apperror.InternalCause(fmt.Errorf("generate otp: %w", otpErr))
		logger.ErrorWithContext(ctx, "Send OTP failed: code generation error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "generate_otp_code",
			"error":   otpErr.Error(),
			"email":   email,
		})
		return err
	}

	if mailErr := uc.mailer.SendOTP(code, email); mailErr != nil {
		observability.ObserveMailerOp("queue_full")
		err = apperror.InternalCause(fmt.Errorf("send otp: %w", mailErr))
		logger.ErrorWithContext(ctx, "Send OTP failed: mailer enqueue error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "mailer_send_otp",
			"error":   mailErr.Error(),
			"email":   email,
		})
		return err
	}

	// store OTP code in Redis and reset failed-attempt counter
	otpKey := fmt.Sprintf("user_otp:%s", email)
	if cacheErr := uc.redisCache.Set(ctx, otpKey, code); cacheErr != nil {
		observability.ObserveCacheOp("redis", "set", "error")
		logger.ErrorWithContext(ctx, "Send OTP: failed to cache OTP code (non-fatal)", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_set_otp",
			"error":   cacheErr.Error(),
			"email":   email,
		})
	} else {
		observability.ObserveCacheOp("redis", "set", "ok")
	}
	_ = uc.redisCache.Del(ctx, otpAttemptsKey(email))

	return nil
}
