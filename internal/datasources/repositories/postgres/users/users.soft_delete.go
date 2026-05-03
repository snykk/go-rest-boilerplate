package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func (r *postgreUserRepository) SoftDelete(ctx context.Context, id string) error {
	const (
		repositoryName = "users"
		funcName       = "SoftDelete"
		queryName      = "softDeleteUser"
		fileName       = "users.soft_delete.go"
	)
	res, err := r.conn.ExecContext(ctx, `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to soft-delete user", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"user_id":    id,
		})
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to read rows-affected after soft delete", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"user_id":    id,
		})
		return err
	}
	if affected == 0 {
		return apperror.NotFound("user not found")
	}
	return nil
}
