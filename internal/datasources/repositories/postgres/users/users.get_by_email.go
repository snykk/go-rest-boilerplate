package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func (r *postgreUserRepository) GetByEmail(ctx context.Context, inDom *domain.User) (outDomain domain.User, err error) {
	const (
		repositoryName = "users"
		funcName       = "GetByEmail"
		queryName      = "selectUserByEmail"
		fileName       = "users.get_by_email.go"
	)
	userRecord := records.FromUsersV1Domain(inDom)

	// Exclude soft-deleted rows — the schema keeps a deleted_at column
	// so "deleted" users remain queryable for audit/restore, but they
	// must not satisfy login or OTP flows.
	err = r.conn.GetContext(ctx, &userRecord, `SELECT * FROM users WHERE "email" = $1 AND deleted_at IS NULL`, userRecord.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, apperror.NotFound("user not found")
		}
		logger.ErrorWithContext(ctx, "Failed to query user by email", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"email":      userRecord.Email,
		})
		return domain.User{}, err
	}

	return userRecord.ToV1Domain(), nil
}
