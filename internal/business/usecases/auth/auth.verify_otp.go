package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
)

// VerifyOTP checks the supplied code against Redis, increments a
// per-email attempt counter, and activates the account on success.
// Lockout fires after Config.OTPMaxAttempts failures — even with the
// correct code, to defeat brute force on the 1M-combination keyspace.
func (uc *usecase) VerifyOTP(ctx context.Context, email, otpCode string) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "VerifyOTP"
		fileName    = "auth.verify_otp.go"
	)
	startTime := time.Now()
	email = domain.NormalizeEmail(email)

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"email":         email,
			"has_otp_code":  otpCode != "",
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
		logger.ErrorWithContext(ctx, "Verify OTP failed: user lookup error", logger.Fields{
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
		logger.ErrorWithContext(ctx, "Verify OTP failed: account already activated", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "check_active",
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return err
	}

	// Brute-force guard. The counter shares the OTP TTL so it
	// resets cleanly on a new SendOTP cycle.
	attemptsKey := otpAttemptsKey(email)
	attempts, incrErr := uc.redisCache.Incr(ctx, attemptsKey)
	if incrErr != nil {
		logger.ErrorWithContext(ctx, "Verify OTP: failed to track attempts (non-fatal)", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_incr_attempts",
			"error":   incrErr.Error(),
			"email":   email,
		})
	} else if attempts == 1 {
		// First attempt in this window — set expiry to match OTP TTL.
		_ = uc.redisCache.Expire(ctx, attemptsKey, uc.cfg.OTPTTL)
	}
	if attempts > int64(uc.cfg.OTPMaxAttempts) {
		err = apperror.Forbidden("too many invalid otp attempts, please request a new code")
		logger.ErrorWithContext(ctx, "Verify OTP failed: lockout (max attempts exceeded)", logger.Fields{
			"usecase":  usecaseName,
			"method":   funcName,
			"file":     fileName,
			"step":     "check_lockout",
			"error":    err.Error(),
			"email":    email,
			"attempts": attempts,
		})
		return err
	}

	otpKey := fmt.Sprintf("user_otp:%s", email)
	otpRedis, getErr := uc.redisCache.Get(ctx, otpKey)
	if getErr != nil {
		observability.ObserveCacheOp("redis", "get", "miss")
		err = apperror.BadRequest("otp code expired or not found")
		logger.ErrorWithContext(ctx, "Verify OTP failed: code expired or not found", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_get_otp",
			"error":   getErr.Error(),
			"email":   email,
		})
		return err
	}
	observability.ObserveCacheOp("redis", "get", "hit")

	// Constant-time compare to defeat per-byte timing attacks on the OTP keyspace.
	if subtle.ConstantTimeCompare([]byte(otpRedis), []byte(otpCode)) != 1 {
		err = apperror.BadRequest("invalid otp code")
		logger.ErrorWithContext(ctx, "Verify OTP failed: invalid code", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "compare_otp",
			"error":   err.Error(),
			"email":   email,
		})
		return err
	}

	// Activate via the User bounded context — auth doesn't touch
	// the user repository directly.
	if activateErr := uc.users.Activate(ctx, user.ID); activateErr != nil {
		err = activateErr
		logger.ErrorWithContext(ctx, "Verify OTP failed: activate error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "users_activate",
			"error":   activateErr.Error(),
			"user_id": user.ID,
		})
		return err
	}

	// cleanup caches
	if delErr := uc.redisCache.Del(ctx, otpKey); delErr != nil {
		logger.ErrorWithContext(ctx, "Verify OTP: failed to delete OTP cache (non-fatal)", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_del_otp",
			"error":   delErr.Error(),
			"email":   email,
		})
	}
	_ = uc.redisCache.Del(ctx, attemptsKey)

	return nil
}
