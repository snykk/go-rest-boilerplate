package players

import (
	"context"
	"fmt"

	"github.com/go-rest-boilerplate/pkg/jwt"
	"github.com/go-rest-boilerplate/internal/apperror"
	"github.com/go-rest-boilerplate/internal/datasources/caches"
	repointerface "github.com/go-rest-boilerplate/internal/datasources/repositories/interface"
	"golang.org/x/sync/singleflight"
	"github.com/go-rest-boilerplate/internal/business/domain"
)

type (
	StoreRequest struct {
		InstallationId string
	}
	StoreResponse struct {
		Player domain.Player
	}

	CreateAnonymousPlayerRequest struct {
		InstallationId string
	}
	CreateAnonymousPlayerResponse struct {
		Player      domain.Player
		AccessToken string
	}
)

type Usecase interface {
	Store(ctx context.Context, req StoreRequest) (StoreResponse, error)
	CreateAnonymousPlayer(ctx context.Context, req CreateAnonymousPlayerRequest) (CreateAnonymousPlayerResponse, error)
}

type Config struct {
	BcryptCost int
}

type usecase struct {
	repo             repointerface.PlayerRepository
	ristrettoCache   caches.RistrettoCache
	cfg              Config
	jwtService 		 jwt.JWTService
	userByEmailGroup singleflight.Group
}

func NewUsecase(repo repointerface.PlayerRepository, ristrettoCache caches.RistrettoCache, jwtService jwt.JWTService, cfg Config) Usecase {
	return &usecase{
		repo:           repo,
		ristrettoCache: ristrettoCache,
		cfg:            cfg,
		jwtService:     jwtService,
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