package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Put stores secret, with its CreatedAt, as its user's only secret of the repository's kind, replacing the one
// the user held and its count of wrong guesses, and reports true. When the held secret was created less than
// cooldown before secret.CreatedAt, it keeps that one, stores nothing and reports false. It decides in one
// statement that waits for a concurrent put for the same user to commit, so of concurrent puts within the
// cooldown only the first is stored. It returns an error when no stored user has the secret's UserID.
func (r *SecretRepo) Put(ctx context.Context, secret domain.Secret, cooldown time.Duration) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`INSERT INTO `+r.table+` (`+kindColumns+`) VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE
		 SET hash = excluded.hash, expires_at = excluded.expires_at, attempts = excluded.attempts,
		     created_at = excluded.created_at
		 WHERE `+r.table+`.created_at <= $6`,
		secret.UserID, secret.Hash, secret.ExpiresAt, secret.Attempts, secret.CreatedAt,
		secret.CreatedAt.Add(-cooldown))
	if err != nil {
		return false, fmt.Errorf("put secret: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
