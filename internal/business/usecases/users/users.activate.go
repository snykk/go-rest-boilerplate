package users

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Activate flips the user's active flag — the only legitimate caller
// is the auth context's VerifyOTP flow. Lives in the User bounded
// context (not Auth) because flipping `active` is an operation on the
// user record, regardless of what triggered it.
func (uc *usecase) Activate(ctx context.Context, userID string) (err error) {
	const (
		usecaseName = "users"
		funcName    = "Activate"
		fileName    = "users.activate.go"
	)
	startTime := time.Now()

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

	u := &domain.User{ID: userID}
	u.Activate() // domain method: sets Active=true and stamps UpdatedAt
	if changeErr := uc.repo.ChangeActiveUser(ctx, u); changeErr != nil {
		err = apperror.InternalCause(fmt.Errorf("activate user: %w", changeErr))
		logger.ErrorWithContext(ctx, "Activate user failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_change_active_user",
			"error":   changeErr.Error(),
			"user_id": userID,
		})
		return err
	}
	// Invalidate the ristretto cache so the next Login doesn't read
	// the stale (Active=false) entry that GetByEmail populated during
	// the OTP flow.
	if existing, getErr := uc.repo.GetByID(ctx, userID); getErr == nil && existing.Email != "" {
		uc.ristrettoCache.Del(fmt.Sprintf("user/%s", existing.Email))
	}
	return nil
}
