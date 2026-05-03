package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func (r *postgreUserRepository) UpdatePassword(ctx context.Context, inDom *domain.User) error {
	const (
		repositoryName = "users"
		funcName       = "UpdatePassword"
		queryName      = "updateUserPassword"
		fileName       = "users.update_password.go"
	)
	userRecord := records.FromUsersV1Domain(inDom)
	res, err := r.conn.NamedExecContext(ctx,
		`UPDATE users SET password = :password, password_changed_at = :password_changed_at, updated_at = :updated_at WHERE id = :id AND deleted_at IS NULL`,
		userRecord)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to update user password", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"user_id":    userRecord.Id,
		})
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to read rows-affected after password update", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"user_id":    userRecord.Id,
		})
		return err
	}
	if affected == 0 {
		return apperror.NotFound("user not found")
	}
	return nil
}
