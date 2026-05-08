package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// ResetPassword consumes a one-shot reset token (from ForgotPassword)
// and replaces the user's password. The token is deleted on success
// so it can't be replayed.
func (uc *usecase) ResetPassword(ctx context.Context, req ResetPasswordRequest) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "ResetPassword"
		fileName    = "auth.reset_password.go"
	)
	startTime := time.Now()
	token := req.Token
	newPassword := req.NewPassword

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"has_token":        token != "",
			"has_new_password": newPassword != "",
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

	if newPassword == "" {
		err = apperror.BadRequest("new password is required")
		logger.ErrorWithContext(ctx, "Reset password failed: empty new password", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "validate_new_password",
			"error":   err.Error(),
		})
		return err
	}
	if token == "" {
		err = apperror.BadRequest("reset token is required")
		logger.ErrorWithContext(ctx, "Reset password failed: empty token", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "validate_token",
			"error":   err.Error(),
		})
		return err
	}

	userID, getErr := uc.redisCache.Get(ctx, resetKey(token))
	if getErr != nil || userID == "" {
		err = apperror.Unauthorized("reset token is invalid or expired")
		fields := logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "redis_get_reset_token",
			"error":   err.Error(),
		}
		if getErr != nil {
			fields["redis_error"] = getErr.Error()
		}
		logger.ErrorWithContext(ctx, "Reset password failed: invalid or expired token", fields)
		return err
	}

	lookupResp, lookupErr := uc.users.GetByID(ctx, users.GetByIDRequest{ID: userID})
	if lookupErr != nil {
		err = lookupErr
		logger.ErrorWithContext(ctx, "Reset password failed: user lookup error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "get_user_by_id",
			"error":   lookupErr.Error(),
			"user_id": userID,
		})
		return err
	}
	user := lookupResp.User
	if hashErr := user.ChangePassword(newPassword, uc.cfg.BcryptCost); hashErr != nil {
		if errors.Is(hashErr, domain.ErrEmptyPassword) {
			err = apperror.BadRequest(hashErr.Error())
		} else {
			err = apperror.InternalCause(fmt.Errorf("hash reset password: %w", hashErr))
		}
		logger.ErrorWithContext(ctx, "Reset password failed: hash error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "domain_change_password",
			"error":   hashErr.Error(),
			"user_id": userID,
		})
		return err
	}
	if updateErr := uc.users.UpdatePassword(ctx, users.UpdatePasswordRequest{User: &user}); updateErr != nil {
		err = updateErr
		logger.ErrorWithContext(ctx, "Reset password failed: update error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "users_update_password",
			"error":   updateErr.Error(),
			"user_id": userID,
		})
		return err
	}
	_ = uc.redisCache.Del(ctx, resetKey(token))
	_ = uc.redisCache.Del(ctx, userResetIndexKey(userID))
	if user.PasswordChangedAt != nil {
		uc.recordTokenCutoff(ctx, userID, *user.PasswordChangedAt)
	}
	return nil
}
