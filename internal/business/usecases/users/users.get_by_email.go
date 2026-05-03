package users

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
)

// GetByEmail returns the user with the given email. The in-memory
// (Ristretto) cache is consulted first; on miss, concurrent goroutines
// share a single DB round-trip via singleflight to prevent a
// thundering herd against Postgres.
func (uc *usecase) GetByEmail(ctx context.Context, email string) (out domain.User, err error) {
	const (
		usecaseName = "users"
		funcName    = "GetByEmail"
		fileName    = "users.get_by_email.go"
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
		if err == nil {
			fields["response"] = logger.Fields{"user_id": out.ID}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	cacheKey := fmt.Sprintf("user/%s", email)
	if val := uc.ristrettoCache.Get(cacheKey); val != nil {
		if cached, ok := val.(domain.User); ok {
			observability.ObserveCacheOp("ristretto", "get", "hit")
			out = cached
			return out, nil
		}
		observability.ObserveCacheOp("ristretto", "get", "error")
		logger.WarnWithContext(ctx, "Get user by email: cache type assertion failed", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "ristretto_cache_get",
			"email":   email,
		})
	} else {
		observability.ObserveCacheOp("ristretto", "get", "miss")
	}

	v, sfErr, _ := uc.userByEmailGroup.Do(email, func() (any, error) {
		user, repoErr := uc.repo.GetByEmail(ctx, &domain.User{Email: email})
		if repoErr != nil {
			return domain.User{}, repoErr
		}
		uc.ristrettoCache.Set(cacheKey, user)
		observability.ObserveCacheOp("ristretto", "set", "ok")
		return user, nil
	})
	if sfErr != nil {
		// Forward typed errors so infra failures don't masquerade as 404.
		err = mapRepoError(sfErr, "get user by email")
		logger.ErrorWithContext(ctx, "Get user by email failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_get_by_email",
			"error":   sfErr.Error(),
			"email":   email,
		})
		return domain.User{}, err
	}
	out = v.(domain.User)
	return out, nil
}
