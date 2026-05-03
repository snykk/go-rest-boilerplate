package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// ChangePassword swaps the authenticated user's password after
// verifying the current one. The new PasswordChangedAt timestamp acts
// as a revocation cutoff — refresh tokens issued before it are
// rejected on /refresh.
func (uc *usecase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) (err error) {
	const (
		usecaseName = "auth"
		funcName    = "ChangePassword"
		fileName    = "auth.change_password.go"
	)
	startTime := time.Now()

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"user_id":              userID,
			"has_current_password": currentPassword != "",
			"has_new_password":     newPassword != "",
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
		logger.ErrorWithContext(ctx, "Change password failed: empty new password", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "validate_new_password",
			"error":   err.Error(),
			"user_id": userID,
		})
		return err
	}
	user, lookupErr := uc.users.GetByID(ctx, userID)
	if lookupErr != nil {
		err = lookupErr
		logger.ErrorWithContext(ctx, "Change password failed: user lookup error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "get_user_by_id",
			"error":   lookupErr.Error(),
			"user_id": userID,
		})
		return err
	}
	if !user.VerifyPassword(currentPassword) {
		err = apperror.Unauthorized("current password is incorrect")
		logger.ErrorWithContext(ctx, "Change password failed: invalid current password", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "verify_current_password",
			"error":   err.Error(),
			"user_id": userID,
		})
		return err
	}
	if hashErr := user.ChangePassword(newPassword, uc.cfg.BcryptCost); hashErr != nil {
		if errors.Is(hashErr, domain.ErrEmptyPassword) {
			err = apperror.BadRequest(hashErr.Error())
		} else {
			err = apperror.InternalCause(fmt.Errorf("hash new password: %w", hashErr))
		}
		logger.ErrorWithContext(ctx, "Change password failed: hash error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "domain_change_password",
			"error":   hashErr.Error(),
			"user_id": userID,
		})
		return err
	}
	if updateErr := uc.users.UpdatePassword(ctx, &user); updateErr != nil {
		err = updateErr
		logger.ErrorWithContext(ctx, "Change password failed: update error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "users_update_password",
			"error":   updateErr.Error(),
			"user_id": userID,
		})
		return err
	}
	return nil
}
