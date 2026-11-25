package postgres

import (
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

const userColumns = "id, email, display_name, password_hash, roles, verified_at, failed_logins, locked_until, " +
	"created_at, updated_at"

type userRow struct {
	ID           string     `db:"id"`
	Email        string     `db:"email"`
	DisplayName  string     `db:"display_name"`
	PasswordHash string     `db:"password_hash"`
	Roles        []string   `db:"roles"`
	VerifiedAt   *time.Time `db:"verified_at"`
	FailedLogins int        `db:"failed_logins"`
	LockedUntil  *time.Time `db:"locked_until"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// toDomain returns the user the row holds. It returns an error when the stored email is not one the domain
// accepts; the error does not repeat the email.
func (r userRow) toDomain() (*domain.User, error) {
	email, err := domain.NewEmail(r.Email)
	if err != nil {
		return nil, fmt.Errorf("read user %s: the stored email is not a valid address", r.ID)
	}
	return domain.Restore(domain.UserStored{
		ID:           r.ID,
		Email:        email,
		DisplayName:  r.DisplayName,
		PasswordHash: r.PasswordHash,
		Roles:        r.Roles,
		VerifiedAt:   inUTC(r.VerifiedAt),
		FailedLogins: r.FailedLogins,
		LockedUntil:  inUTC(r.LockedUntil),
		CreatedAt:    r.CreatedAt.UTC(),
		UpdatedAt:    r.UpdatedAt.UTC(),
	}), nil
}

// userRowOf returns the row that stores user. A user with no roles gets an empty list, never NULL.
func userRowOf(user *domain.User) userRow {
	stored := user.Stored()
	roles := stored.Roles
	if roles == nil {
		roles = []string{}
	}
	return userRow{
		ID:           stored.ID,
		Email:        stored.Email.String(),
		DisplayName:  stored.DisplayName,
		PasswordHash: stored.PasswordHash,
		Roles:        roles,
		VerifiedAt:   stored.VerifiedAt,
		FailedLogins: stored.FailedLogins,
		LockedUntil:  stored.LockedUntil,
		CreatedAt:    stored.CreatedAt,
		UpdatedAt:    stored.UpdatedAt,
	}
}

func inUTC(moment *time.Time) *time.Time {
	if moment == nil {
		return nil
	}
	utc := moment.UTC()
	return &utc
}
