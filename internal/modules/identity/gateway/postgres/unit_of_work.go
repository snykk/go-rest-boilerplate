package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

// UnitOfWork runs units of work on the identity tables. It is safe for concurrent use when it runs on a pool.
type UnitOfWork struct {
	db database.Querier
}

var _ usecase.UnitOfWork = (*UnitOfWork)(nil)

// NewUnitOfWork returns a UnitOfWork that runs each unit in a transaction on querier: on a pool, a transaction
// of its own; on a transaction, a savepoint.
func NewUnitOfWork(querier database.Querier) *UnitOfWork {
	return &UnitOfWork{db: querier}
}

// Do calls work with every identity repository built on one transaction. It commits when work returns nil,
// and rolls back and returns work's error when it does not. It returns an error when the transaction cannot
// begin or commit.
func (u *UnitOfWork) Do(ctx context.Context, work func(repositories usecase.Repositories) error) error {
	return pgx.BeginFunc(ctx, u.db, func(transaction pgx.Tx) error {
		return work(usecase.Repositories{
			Users:         NewUserRepo(transaction),
			Verifications: NewVerificationRepo(transaction),
			Resets:        NewResetRepo(transaction),
			Sessions:      NewSessionRepo(transaction),
		})
	})
}
