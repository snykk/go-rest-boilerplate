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

func (r *postgreUserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	const (
		repositoryName = "users"
		funcName       = "GetByID"
		queryName      = "selectUserByID"
		fileName       = "users.get_by_id.go"
	)
	var stored records.Users
	err := r.conn.GetContext(ctx, &stored, `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, apperror.NotFound("user not found")
		}
		logger.ErrorWithContext(ctx, "Failed to query user by id", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
			"user_id":    id,
		})
		return domain.User{}, err
	}
	return stored.ToV1Domain(), nil
}
