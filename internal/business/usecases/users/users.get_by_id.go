package users

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// GetByID returns the user with the given primary key. ID lookups
// don't go through the email-keyed cache — they're rare enough that
// a direct DB read is fine, and caching by ID would just duplicate
// state without a measurable hit rate.
func (uc *usecase) GetByID(ctx context.Context, id string) (out domain.User, err error) {
	const (
		usecaseName = "users"
		funcName    = "GetByID"
		fileName    = "users.get_by_id.go"
	)
	startTime := time.Now()

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"user_id": id,
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

	user, repoErr := uc.repo.GetByID(ctx, id)
	if repoErr != nil {
		err = mapRepoError(repoErr, "get user by id")
		logger.ErrorWithContext(ctx, "Get user by id failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_get_by_id",
			"error":   repoErr.Error(),
			"user_id": id,
		})
		return domain.User{}, err
	}
	out = user
	return out, nil
}
