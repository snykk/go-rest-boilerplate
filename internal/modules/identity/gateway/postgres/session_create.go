package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Create stores session, live until it expires. It returns an error when no stored user has the session's
// UserID.
func (r *SessionRepo) Create(ctx context.Context, session domain.Session) error {
	return insertSession(ctx, r.db, session)
}
