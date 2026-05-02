package users

import (
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	repointerface "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/interface"
	"golang.org/x/sync/singleflight"
)

// Config carries tunables that the usecase forwards into the domain
// layer. Domain itself takes bcryptCost as a parameter so it can stay
// free of configuration concerns; the usecase is the boundary that
// knows about config.
type Config struct {
	BcryptCost int
}

// usecase carries the dependencies and any cross-method state. Each
// method lives in its own file so PR diffs stay surgical when a
// single behavior changes.
type usecase struct {
	repo           repointerface.UserRepository
	ristrettoCache caches.RistrettoCache
	cfg            Config

	// userByEmailGroup coalesces concurrent cache misses for the
	// same email so a thundering herd can't fan out into N parallel
	// DB round-trips. The group is keyed by normalized email.
	userByEmailGroup singleflight.Group
}

// NewUsecase builds the User CRUD use case. It does not depend on any
// auth-related collaborator (no JWT, no Redis, no mailer) — that's
// the whole point of the User vs Auth split.
func NewUsecase(repo repointerface.UserRepository, ristrettoCache caches.RistrettoCache, cfg Config) Usecase {
	return &usecase{
		repo:           repo,
		ristrettoCache: ristrettoCache,
		cfg:            cfg,
	}
}

// mapRepoError preserves DomainError types returned from the
// repository while wrapping raw errors in a formatted internal error.
// Without this, errors.As(err, *DomainError) upstream would fail.
func mapRepoError(err error, op string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*apperror.DomainError); ok {
		return err
	}
	return apperror.InternalCause(fmt.Errorf("%s: %w", op, err))
}
