package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Revoke revokes at now the family of the session with the given token hash: every session of the sign-in the
// token belongs to, across its rotations, that is not revoked yet. It holds the user's row lock while it does,
// as Rotate does, so a rotation of a token of the family either finishes first, and the session it stored is
// revoked too, or starts after and finds its token revoked. It runs in a transaction of its own, or in a
// savepoint when the repository runs on a transaction. It returns nil when no session has that hash.
func (r *SessionRepo) Revoke(ctx context.Context, tokenHash string, now time.Time) error {
	return pgx.BeginFunc(ctx, r.db, func(transaction pgx.Tx) error {
		family, found, err := lockOwner(ctx, transaction, tokenHash)
		if err != nil || !found {
			return err
		}

		return revokeFamily(ctx, transaction, family, now)
	})
}
