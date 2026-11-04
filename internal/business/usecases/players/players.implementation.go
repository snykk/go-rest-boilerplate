package players

import (
	"context"
	"fmt"

	"github.com/go-rest-boilerplate/internal/apperror"
	"github.com/go-rest-boilerplate/internal/datasources/caches"
	repointerface "github.com/go-rest-boilerplate/internal/datasources/repositories/interface"
	"golang.org/x/sync/singleflight"
	"github.com/go-rest-boilerplate/internal/business/domain"
)

type (
	CreateAnonymousPlayerRequest struct {
		InstallationID string
	}
	CreateAnonymousPlayerResponse struct {
		Player domain.Player
	}
)

type Usecase interface {
	CreateAnonymousPlayer(ctx context.Context, req CreateAnonymousPlayerRequest) (CreateAnonymousPlayerResponse, error)
}

type Config struct {
	BcryptCost int
}

type usecase struct {
	repo             repointerface.UserRepository
	ristrettoCache   caches.RistrettoCache
	cfg              Config
	userByEmailGroup singleflight.Group
}

func NewUsecase(repo repointerface.UserRepository, ristrettoCache caches.RistrettoCache, cfg Config) Usecase {
	return &usecase{
		repo:           repo,
		ristrettoCache: ristrettoCache,
		cfg:            cfg,
	}
}

func mapRepoError(err error, op string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*apperror.DomainError); ok {
		return err
	}
	return apperror.InternalCause(fmt.Errorf("%s: %w", op, err))
}