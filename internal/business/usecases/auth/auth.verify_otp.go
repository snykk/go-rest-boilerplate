package auth

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
)

// VerifyOTP checks the supplied code against Redis, increments a
// per-email attempt counter, and activates the account on success.
// Lockout fires after Config.OTPMaxAttempts failures — even with the
// correct code, to defeat brute force on the 1M-combination keyspace.
func (uc *usecase) VerifyOTP(ctx context.Context, email, otpCode string) error {
	email = domain.NormalizeEmail(email)
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return apperror.NotFound("email not found")
	}

	if user.Active {
		return apperror.BadRequest("account already activated")
	}

	// Brute-force guard. The counter shares the OTP TTL so it
	// resets cleanly on a new SendOTP cycle.
	attemptsKey := otpAttemptsKey(email)
	attempts, err := uc.redisCache.Incr(ctx, attemptsKey)
	if err != nil {
		logger.Error("failed to track OTP attempts", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	} else if attempts == 1 {
		// First attempt in this window — set expiry to match OTP TTL.
		_ = uc.redisCache.Expire(ctx, attemptsKey, uc.cfg.OTPTTL)
	}
	if attempts > int64(uc.cfg.OTPMaxAttempts) {
		return apperror.Forbidden("too many invalid otp attempts, please request a new code")
	}

	otpKey := fmt.Sprintf("user_otp:%s", email)
	otpRedis, err := uc.redisCache.Get(ctx, otpKey)
	if err != nil {
		observability.ObserveCacheOp("redis", "get", "miss")
		return apperror.BadRequest("otp code expired or not found")
	}
	observability.ObserveCacheOp("redis", "get", "hit")

	if otpRedis != otpCode {
		return apperror.BadRequest("invalid otp code")
	}

	// Activate via the User bounded context — auth doesn't touch
	// the user repository directly.
	if err = uc.users.Activate(ctx, user.ID); err != nil {
		return err
	}

	// cleanup caches
	if err = uc.redisCache.Del(ctx, otpKey); err != nil {
		logger.Error("failed to delete OTP cache", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	}
	_ = uc.redisCache.Del(ctx, attemptsKey)

	return nil
}
