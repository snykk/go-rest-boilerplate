package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Rotate spends the session with the given token hash, when it is live at now, stores next with the spent
// session's user and family as its successor, and returns that user. It holds the user's row lock throughout,
// as Revoke, RevokeAll and UserRepo.Update do, so concurrent rotations of one token succeed once, and a rotation
// and a Revoke or RevokeAll for the same user apply one after the other. It runs in a transaction of its own, or
// in a savepoint when the repository runs on a transaction.
//
// It returns domain.ErrInvalidToken, changing nothing, for an unknown or revoked token, for a token spent less
// than grace before now, and for an expired token never spent. For a token spent grace or more before now,
// expired since or not, it revokes the token's whole family at now and returns the token's user with
// domain.ErrTokenReused; the token is revoked with its family, so presenting it again returns
// domain.ErrInvalidToken.
func (r *SessionRepo) Rotate(
	ctx context.Context, tokenHash string, next domain.Session, now time.Time, grace time.Duration,
) (string, error) {
	var userID string
	var reused bool
	err := pgx.BeginFunc(ctx, r.db, func(transaction pgx.Tx) error {
		var err error
		userID, reused, err = rotate(ctx, transaction, tokenHash, next, now, grace)
		return err
	})
	switch {
	case err != nil:
		return "", err
	case reused:
		return userID, domain.ErrTokenReused
	case userID == "":
		return "", domain.ErrInvalidToken
	}
	return userID, nil
}

// rotate spends, in transaction, the live session with the token hash, stores next in its place and returns its user.
// For a token that is not live it returns no error, so the transaction keeps what it changed: for a token spent
// grace or more before now, the revoked family, and the token's user with reused true; otherwise nothing, and
// no user.
func rotate(
	ctx context.Context, transaction pgx.Tx, tokenHash string, next domain.Session, now time.Time, grace time.Duration,
) (userID string, reused bool, err error) {
	_, found, err := lockOwner(ctx, transaction, tokenHash)
	if err != nil || !found {
		return "", false, err
	}

	var family string
	err = transaction.QueryRow(ctx,
		`UPDATE sessions SET spent_at = $2
		  WHERE token_hash = $1 AND spent_at IS NULL AND revoked_at IS NULL AND expires_at > $2
		  RETURNING user_id, family`,
		tokenHash, now).Scan(&userID, &family)
	if errors.Is(err, pgx.ErrNoRows) {
		return revokeIfReused(ctx, transaction, tokenHash, now, grace)
	}
	if err != nil {
		return "", false, fmt.Errorf("spend session: %w", err)
	}

	next.UserID, next.Family = userID, family
	if err := insertSession(ctx, transaction, next); err != nil {
		return "", false, err
	}
	return userID, false, nil
}

// revokeIfReused revokes at now the family of the session with the token hash when that session is not
// revoked and was spent grace or more before now: its token came back after its successor was handed out, so
// someone who should not hold it may have it. It then returns the session's user and reused true; otherwise
// it changes nothing and returns no user.
func revokeIfReused(
	ctx context.Context, transaction pgx.Tx, tokenHash string, now time.Time, grace time.Duration,
) (userID string, reused bool, err error) {
	var family string
	var spentAt, revokedAt *time.Time
	err = transaction.QueryRow(ctx,
		`SELECT user_id, family, spent_at, revoked_at FROM sessions WHERE token_hash = $1`, tokenHash).
		Scan(&userID, &family, &spentAt, &revokedAt)
	if err != nil {
		return "", false, fmt.Errorf("read session: %w", err)
	}
	if revokedAt != nil || spentAt == nil || spentAt.After(now.Add(-grace)) {
		return "", false, nil
	}

	if err := revokeFamily(ctx, transaction, family, now); err != nil {
		return "", false, err
	}
	return userID, true, nil
}
