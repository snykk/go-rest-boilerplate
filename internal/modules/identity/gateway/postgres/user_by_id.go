package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ByID returns the user with the given ID. It returns domain.ErrNotFound when there is none, including when
// userID is not a UUID.
func (r *UserRepo) ByID(ctx context.Context, userID string) (*domain.User, error) {
	return queryUser(ctx, r.db, "select user", `SELECT `+userColumns+` FROM users WHERE id = $1`, userID)
}
