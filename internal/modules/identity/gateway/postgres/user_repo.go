// Package postgres stores the identity module's users, sessions, verification codes and password reset tokens
// in the PostgreSQL tables created by the identity migrations.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

const usersEmailIndex = "users_email_key"

// UserRepo stores users in the users table, with at most one user per email in any case. Times are read back
// in UTC.
type UserRepo struct {
	db database.Querier
}

var _ usecase.UserRepository = (*UserRepo)(nil)

// NewUserRepo returns a UserRepo that runs its queries on querier. On a pool each call stands alone; on a
// transaction every call commits or rolls back with it, which is how a unit of work groups repositories.
func NewUserRepo(querier database.Querier) *UserRepo {
	return &UserRepo{db: querier}
}

// queryUser returns the one user that query, run on querier with args, selects. It returns domain.ErrNotFound
// when the query selects none or is given an ID that is not a UUID.
func queryUser(
	ctx context.Context, querier database.Querier, operation, query string, args ...any,
) (*domain.User, error) {
	rows, err := querier.Query(ctx, query, args...)
	if err != nil {
		return nil, translateUser(err, operation)
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, translateUser(err, operation)
	}
	return row.toDomain()
}

func translateUser(err error, operation string) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows), malformedID(err):
		return domain.ErrNotFound
	case violatesUnique(err, usersEmailIndex):
		return domain.ErrEmailTaken
	}
	return fmt.Errorf("%s: %w", operation, err)
}
