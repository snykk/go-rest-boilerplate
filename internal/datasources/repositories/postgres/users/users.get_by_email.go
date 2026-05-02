package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

func (r *postgreUserRepository) GetByEmail(ctx context.Context, inDom *domain.User) (outDomain domain.User, err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	// Exclude soft-deleted rows — the schema keeps a deleted_at column
	// so "deleted" users remain queryable for audit/restore, but they
	// must not satisfy login or OTP flows.
	err = r.conn.GetContext(ctx, &userRecord, `SELECT * FROM users WHERE "email" = $1 AND deleted_at IS NULL`, userRecord.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, apperror.NotFound("user not found")
		}
		return domain.User{}, err
	}

	return userRecord.ToV1Domain(), nil
}
