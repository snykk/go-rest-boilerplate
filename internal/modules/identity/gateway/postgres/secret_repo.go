package postgres

import (
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

const (
	verificationsTable = "email_verifications"
	resetsTable        = "password_resets"
)

// SecretRepo stores the secrets of one kind, e-mail verification codes or password reset tokens, in that
// kind's table, at most one per user. The two kinds never meet. Times are read back in UTC.
type SecretRepo struct {
	db    database.Querier
	table string
}

var _ usecase.SecretRepository = (*SecretRepo)(nil)

// NewVerificationRepo returns a SecretRepo of e-mail verification codes that runs its queries on querier. On a
// pool each call stands alone; on a transaction every call commits or rolls back with it, which is how a unit
// of work groups repositories.
func NewVerificationRepo(querier database.Querier) *SecretRepo {
	return &SecretRepo{db: querier, table: verificationsTable}
}

// NewResetRepo returns a SecretRepo of password reset tokens that runs its queries on querier, on a pool or on
// a transaction as NewVerificationRepo does.
func NewResetRepo(querier database.Querier) *SecretRepo {
	return &SecretRepo{db: querier, table: resetsTable}
}
