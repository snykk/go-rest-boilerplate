package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
)

func (r *postgreUserRepository) SoftDelete(ctx context.Context, id string) error {
	res, err := r.conn.ExecContext(ctx, `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
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
