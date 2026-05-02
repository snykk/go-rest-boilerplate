package postgres

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

func (r *postgreUserRepository) ChangeActiveUser(ctx context.Context, inDom *domain.User) (err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `UPDATE users SET active = :active, updated_at = NOW() WHERE id = :id AND deleted_at IS NULL`, userRecord)

	return
}
