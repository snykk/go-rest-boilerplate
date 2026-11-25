package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// RevokeAll revokes at now every session of the user that is live at now, in every family. It holds the
// user's row lock while it does, as Rotate does, so a rotation of one of the user's tokens either finishes
// first, and the session it stored is revoked too, or starts after and finds its token revoked. It runs in a
// transaction of its own, or in a savepoint when the repository runs on a transaction. It returns nil for a
// user with no session, and an error when userID is not a UUID.
func (r *SessionRepo) RevokeAll(ctx context.Context, userID string, now time.Time) error {
	return pgx.BeginFunc(ctx, r.db, func(transaction pgx.Tx) error {
		if err := lockUser(ctx, transaction, userID); err != nil {
			return err
		}

		_, err := transaction.Exec(ctx,
			`UPDATE sessions SET revoked_at = $2
			  WHERE user_id = $1 AND spent_at IS NULL AND revoked_at IS NULL AND expires_at > $2`,
			userID, now)
		if err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		return nil
	})
}

// lockUser takes, in transaction, the row lock of the user with the given ID, the lock lockOwner takes for a session's
// user. It takes nothing for an unknown user.
func lockUser(ctx context.Context, transaction pgx.Tx, userID string) error {
	_, err := transaction.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE`, userID)
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	return nil
}
