package users

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

// GetByID returns the user with the given primary key. ID lookups
// don't go through the email-keyed cache — they're rare enough that
// a direct DB read is fine, and caching by ID would just duplicate
// state without a measurable hit rate.
func (uc *usecase) GetByID(ctx context.Context, id string) (entities.UserDomain, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entities.UserDomain{}, mapRepoError(err, "get user by id")
	}
	return user, nil
}
