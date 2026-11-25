package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

// SessionRepo stores refresh-token sessions in the sessions table. A session is live while it is neither
// spent nor revoked and expires after the time it is asked about. Every change to a user's sessions, except
// storing a new one, first takes the user's row lock, the lock UserRepo.Update holds while it changes the
// user, so such changes apply one after the other.
type SessionRepo struct {
	db database.Querier
}

var _ usecase.SessionRepository = (*SessionRepo)(nil)

// NewSessionRepo returns a SessionRepo that runs its queries on querier. On a pool each call stands alone; on a
// transaction every call commits or rolls back with it, which is how a unit of work groups repositories.
func NewSessionRepo(querier database.Querier) *SessionRepo {
	return &SessionRepo{db: querier}
}

func insertSession(ctx context.Context, querier database.Querier, session domain.Session) error {
	_, err := querier.Exec(ctx,
		`INSERT INTO sessions (id, user_id, family, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		session.ID, session.UserID, session.Family, session.TokenHash, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// lockOwner takes, in transaction, the row lock of the user who holds the session with the token hash, and returns the
// session's family. It reports false when no session has that hash.
func lockOwner(ctx context.Context, transaction pgx.Tx, tokenHash string) (family string, found bool, err error) {
	err = transaction.QueryRow(ctx,
		`SELECT sessions.family FROM users JOIN sessions ON sessions.user_id = users.id
		  WHERE sessions.token_hash = $1
		  FOR NO KEY UPDATE OF users`,
		tokenHash).Scan(&family)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lock the user of the session: %w", err)
	}
	return family, true, nil
}

// revokeFamily revokes at now, in transaction, every session of family that is not revoked yet.
func revokeFamily(ctx context.Context, transaction pgx.Tx, family string, now time.Time) error {
	_, err := transaction.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE family = $1 AND revoked_at IS NULL`,
		family, now)
	if err != nil {
		return fmt.Errorf("revoke session family: %w", err)
	}
	return nil
}
