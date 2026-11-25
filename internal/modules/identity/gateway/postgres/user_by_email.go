package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ByEmail returns the user with the given email, compared in any case, or domain.ErrNotFound.
func (r *UserRepo) ByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	return queryUser(ctx, r.db, "select user by email",
		`SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1)`, email.String())
}
