package users

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// UpdatePassword persists the already-hashed password + revocation
// timestamp on the supplied user. Caller is expected to invoke
// domain.User.ChangePassword first to populate Password +
// PasswordChangedAt + UpdatedAt before calling this.
func (uc *usecase) UpdatePassword(ctx context.Context, req UpdatePasswordRequest) (err error) {
	const (
		usecaseName = "users"
		funcName    = "UpdatePassword"
		fileName    = "users.update_password.go"
	)
	startTime := time.Now()

	var userID string
	if req.User != nil {
		userID = req.User.ID
	}
	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"user_id": userID,
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

	if req.User == nil || req.User.ID == "" {
		err = apperror.BadRequest("user id required")
		logger.ErrorWithContext(ctx, "Update password failed: missing user id", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "validate_input",
			"error":   err.Error(),
		})
		return err
	}
	if repoErr := uc.repo.UpdatePassword(ctx, req.User); repoErr != nil {
		err = mapRepoError(repoErr, fmt.Sprintf("update password for %s", req.User.ID))
		logger.ErrorWithContext(ctx, "Update password failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_update_password",
			"error":   repoErr.Error(),
			"user_id": req.User.ID,
		})
		return err
	}
	// Invalidate the email-keyed ristretto entry so the next Login
	// doesn't read the stale (old-hash) cached user.
	email := req.User.Email
	if email == "" {
		if existing, getErr := uc.repo.GetByID(ctx, req.User.ID); getErr == nil {
			email = existing.Email
		}
	}
	if email != "" {
		uc.ristrettoCache.Del(fmt.Sprintf("user/%s", email))
	}
	return nil
}
