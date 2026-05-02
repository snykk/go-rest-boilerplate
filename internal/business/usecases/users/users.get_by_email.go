package users

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

// GetByEmail returns the user with the given email. The in-memory
// (Ristretto) cache is consulted first; on miss, concurrent goroutines
// share a single DB round-trip via singleflight to prevent a
// thundering herd against Postgres.
func (uc *usecase) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	email = domain.NormalizeEmail(email)
	cacheKey := fmt.Sprintf("user/%s", email)
	if val := uc.ristrettoCache.Get(cacheKey); val != nil {
		if cached, ok := val.(domain.User); ok {
			observability.ObserveCacheOp("ristretto", "get", "hit")
			return cached, nil
		}
		observability.ObserveCacheOp("ristretto", "get", "error")
		logger.Info("cache type assertion failed, fetching from DB", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryCache})
	} else {
		observability.ObserveCacheOp("ristretto", "get", "miss")
	}

	v, err, _ := uc.userByEmailGroup.Do(email, func() (any, error) {
		user, repoErr := uc.repo.GetByEmail(ctx, &domain.User{Email: email})
		if repoErr != nil {
			return domain.User{}, repoErr
		}
		uc.ristrettoCache.Set(cacheKey, user)
		observability.ObserveCacheOp("ristretto", "set", "ok")
		return user, nil
	})
	if err != nil {
		return domain.User{}, apperror.NotFound("email not found")
	}
	return v.(domain.User), nil
}
