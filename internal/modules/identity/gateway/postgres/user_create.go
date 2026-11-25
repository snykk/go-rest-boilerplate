package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Create inserts user. It returns domain.ErrEmailTaken when another user has the same email in any case.
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	row := userRowOf(user)
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (`+userColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		row.ID, row.Email, row.DisplayName, row.PasswordHash, row.Roles, row.VerifiedAt, row.FailedLogins,
		row.LockedUntil, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return translateUser(err, "insert user")
	}
	return nil
}
