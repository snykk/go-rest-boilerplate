package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

func (r *postgreUserRepository) UpdatePassword(ctx context.Context, inDom *domain.User) error {
	userRecord := records.FromUsersV1Domain(inDom)
	res, err := r.conn.NamedExecContext(ctx,
		`UPDATE users SET password = :password, password_changed_at = :password_changed_at, updated_at = :updated_at WHERE id = :id AND deleted_at IS NULL`,
		userRecord)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperror.NotFound("user not found")
	}
	return nil
}
