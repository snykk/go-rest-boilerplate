package users

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
)

// Store hashes the password, normalizes the email, stamps CreatedAt,
// and inserts the user. The repo's INSERT … RETURNING gives us the
// persisted row in one round-trip so the caller gets the database-
// generated ID without a follow-up read.
func (uc *usecase) Store(ctx context.Context, in *entities.UserDomain) (entities.UserDomain, error) {
	hashed, err := helpers.GenerateHash(in.Password)
	if err != nil {
		return entities.UserDomain{}, apperror.InternalCause(fmt.Errorf("hash password: %w", err))
	}
	in.Password = hashed
	in.Email = normalizeEmail(in.Email)
	in.CreatedAt = time.Now().In(constants.GMT7)

	stored, err := uc.repo.Store(ctx, in)
	if err != nil {
		return entities.UserDomain{}, mapRepoError(err, "store user")
	}
	return stored, nil
}
