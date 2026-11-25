package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Update applies change to the user with the given ID and saves every field while it holds the user's row
// lock, so concurrent updates of one user apply one after the other. SessionRepo.Rotate and
// SessionRepo.RevokeAll take the same lock before they touch the user's sessions. Update runs in a transaction
// of its own, or in a savepoint when the repository runs on a transaction. It returns the saved user;
// domain.ErrNotFound when there is none, including when userID is not a UUID; change's error, with nothing
// saved; and domain.ErrEmailTaken when another user has the new email.
func (r *UserRepo) Update(
	ctx context.Context, userID string, change func(user *domain.User) error,
) (*domain.User, error) {
	var user *domain.User
	err := pgx.BeginFunc(ctx, r.db, func(transaction pgx.Tx) error {
		var err error
		user, err = queryUser(ctx, transaction, "lock user",
			`SELECT `+userColumns+` FROM users WHERE id = $1 FOR NO KEY UPDATE`, userID)
		if err != nil {
			return err
		}

		if err := change(user); err != nil {
			return err
		}

		saved := userRowOf(user)
		_, err = transaction.Exec(ctx,
			`UPDATE users
			    SET email = $2, display_name = $3, password_hash = $4, roles = $5, verified_at = $6,
			        failed_logins = $7, locked_until = $8, updated_at = $9
			  WHERE id = $1`,
			saved.ID, saved.Email, saved.DisplayName, saved.PasswordHash, saved.Roles, saved.VerifiedAt,
			saved.FailedLogins, saved.LockedUntil, saved.UpdatedAt)
		if err != nil {
			return translateUser(err, "update user")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}
