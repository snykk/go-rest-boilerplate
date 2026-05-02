package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

func (r *postgreUserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	var stored records.Users
	err := r.conn.GetContext(ctx, &stored, `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, apperror.NotFound("user not found")
		}
		return domain.User{}, err
	}
	return stored.ToV1Domain(), nil
}
