package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Store builds a fresh domain.User (which normalizes email, hashes
// password, and stamps CreatedAt — all in one place) and inserts it.
// The repo's INSERT … RETURNING gives us the persisted row in a
// single round-trip so the caller gets the database-generated ID
// without a follow-up read.
//
// Note that the input *domain.User is treated as a DTO of registration
// fields; we don't mutate it and we don't trust its hash/CreatedAt —
// domain.NewUser is the only path that produces a valid User.
func (uc *usecase) Store(ctx context.Context, in *domain.User) (out domain.User, err error) {
	const (
		usecaseName = "users"
		funcName    = "Store"
		fileName    = "users.store.go"
	)
	startTime := time.Now()

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"username": in.Username,
			"email":    in.Email,
			"role_id":  in.RoleID,
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
		if err == nil {
			fields["response"] = logger.Fields{"user_id": out.ID, "email": out.Email}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	user, buildErr := domain.NewUser(in.Username, in.Email, in.Password, in.RoleID, uc.cfg.BcryptCost)
	if buildErr != nil {
		// Domain validation errors (empty fields) are user-facing —
		// surface them as BadRequest. Anything else (e.g. bcrypt
		// failure) is an internal fault.
		if errors.Is(buildErr, domain.ErrEmptyUsername) ||
			errors.Is(buildErr, domain.ErrEmptyEmail) ||
			errors.Is(buildErr, domain.ErrInvalidEmail) ||
			errors.Is(buildErr, domain.ErrEmptyPassword) {
			err = apperror.BadRequest(buildErr.Error())
		} else {
			err = apperror.InternalCause(fmt.Errorf("build user: %w", buildErr))
		}
		logger.ErrorWithContext(ctx, "Store user failed: build error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "domain_new_user",
			"error":   buildErr.Error(),
			"email":   in.Email,
		})
		return domain.User{}, err
	}

	stored, repoErr := uc.repo.Store(ctx, user)
	if repoErr != nil {
		err = mapRepoError(repoErr, "store user")
		logger.ErrorWithContext(ctx, "Store user failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_store",
			"error":   repoErr.Error(),
			"email":   user.Email,
		})
		return domain.User{}, err
	}
	out = stored
	return out, nil
}
