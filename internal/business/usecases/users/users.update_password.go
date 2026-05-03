package users

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// UpdatePassword persists the already-hashed password + revocation
// timestamp on the supplied user. Caller is expected to invoke
// domain.User.ChangePassword first to populate Password +
// PasswordChangedAt + UpdatedAt before calling this.
func (uc *usecase) UpdatePassword(ctx context.Context, user *domain.User) (err error) {
	const (
		usecaseName = "users"
		funcName    = "UpdatePassword"
		fileName    = "users.update_password.go"
	)
	startTime := time.Now()

	var userID string
	if user != nil {
		userID = user.ID
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

	if user == nil || user.ID == "" {
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
	if repoErr := uc.repo.UpdatePassword(ctx, user); repoErr != nil {
		err = mapRepoError(repoErr, fmt.Sprintf("update password for %s", user.ID))
		logger.ErrorWithContext(ctx, "Update password failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_update_password",
			"error":   repoErr.Error(),
			"user_id": user.ID,
		})
		return err
	}
	// Invalidate the email-keyed ristretto entry so the next Login
	// doesn't read the stale (old-hash) cached user. We need the
	// email; if the caller didn't populate it on the User struct
	// (just userID + new password), fetch it from the repo.
	email := user.Email
	if email == "" {
		if existing, getErr := uc.repo.GetByID(ctx, user.ID); getErr == nil {
			email = existing.Email
		}
	}
	if email != "" {
		uc.ristrettoCache.Del(fmt.Sprintf("user/%s", email))
	}
	return nil
}
