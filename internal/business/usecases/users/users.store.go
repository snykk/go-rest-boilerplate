package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
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
func (uc *usecase) Store(ctx context.Context, in *domain.User) (domain.User, error) {
	user, err := domain.NewUser(in.Username, in.Email, in.Password, in.RoleID, uc.cfg.BcryptCost)
	if err != nil {
		// Domain validation errors (empty fields) are user-facing —
		// surface them as BadRequest. Anything else (e.g. bcrypt
		// failure) is an internal fault.
		if errors.Is(err, domain.ErrEmptyUsername) ||
			errors.Is(err, domain.ErrEmptyEmail) ||
			errors.Is(err, domain.ErrInvalidEmail) ||
			errors.Is(err, domain.ErrEmptyPassword) {
			return domain.User{}, apperror.BadRequest(err.Error())
		}
		return domain.User{}, apperror.InternalCause(fmt.Errorf("build user: %w", err))
	}

	stored, err := uc.repo.Store(ctx, user)
	if err != nil {
		return domain.User{}, mapRepoError(err, "store user")
	}
	return stored, nil
}
