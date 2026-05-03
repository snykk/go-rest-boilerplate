package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func (r *postgreUserRepository) ChangeActiveUser(ctx context.Context, inDom *domain.User) (err error) {
	const (
		repositoryName = "users"
		funcName       = "ChangeActiveUser"
		queryName      = "updateUserActive"
		fileName       = "users.change_active_user.go"
	)
	userRecord := records.FromUsersV1Domain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `UPDATE users SET active = :active, updated_at = NOW() WHERE id = :id AND deleted_at IS NULL`, userRecord)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to update user active flag", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"user_id":    userRecord.Id,
		})
	}
	return
}
