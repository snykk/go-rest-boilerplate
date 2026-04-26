package users

import (
	"fmt"
	"strings"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/repositories"
	"golang.org/x/sync/singleflight"
)

// usecase carries the dependencies and any cross-method state. Each
// method lives in its own file so PR diffs stay surgical when a
// single behavior changes.
type usecase struct {
	repo           repositories.UserRepository
	ristrettoCache caches.RistrettoCache

	// userByEmailGroup coalesces concurrent cache misses for the
	// same email so a thundering herd can't fan out into N parallel
	// DB round-trips. The group is keyed by normalized email.
	userByEmailGroup singleflight.Group
}

// NewUsecase builds the User CRUD use case. It does not depend on any
// auth-related collaborator (no JWT, no Redis, no mailer) — that's
// the whole point of the User vs Auth split.
func NewUsecase(repo repositories.UserRepository, ristrettoCache caches.RistrettoCache) Usecase {
	return &usecase{
		repo:           repo,
		ristrettoCache: ristrettoCache,
	}
}

// normalizeEmail trims whitespace and lowercases the address so
// "User@Example.com " and "user@example.com" hash to the same Redis
// key, query the same DB row, and produce the same uniqueness
// violation. RFC 5321 says the local part is technically
// case-sensitive, but every consumer-grade mail provider treats it
// case-insensitively; matching that expectation avoids "I can't log
// in because I capitalized the U" support tickets.
func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

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
