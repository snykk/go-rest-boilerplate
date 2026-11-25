package postgres

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// kindColumns are the columns of the table of each kind of secret.
const kindColumns = "user_id, hash, expires_at, attempts, created_at"

type secretRow struct {
	UserID    string    `db:"user_id"`
	Hash      string    `db:"hash"`
	ExpiresAt time.Time `db:"expires_at"`
	Attempts  int       `db:"attempts"`
	CreatedAt time.Time `db:"created_at"`
}

func (r secretRow) toDomain() domain.Secret {
	return domain.Secret{
		UserID:    r.UserID,
		Hash:      r.Hash,
		ExpiresAt: r.ExpiresAt.UTC(),
		Attempts:  r.Attempts,
		CreatedAt: r.CreatedAt.UTC(),
	}
}
