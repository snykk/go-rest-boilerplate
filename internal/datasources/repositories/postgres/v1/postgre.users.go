package v1

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

type postgreUserRepository struct {
	conn *sqlx.DB
}

func NewUserRepository(conn *sqlx.DB) V1Domains.UserRepository {
	return &postgreUserRepository{
		conn: conn,
	}
}

func (r *postgreUserRepository) Store(ctx context.Context, inDom *V1Domains.UserDomain) (err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `INSERT INTO users(id, username, email, password, active, role_id, created_at) VALUES (uuid_generate_v4(), :username, :email, :password, false, :role_id, :created_at)`, userRecord)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return constants.ErrConflict("username or email already exists")
		}
		return err
	}

	return nil
}

func (r *postgreUserRepository) GetByEmail(ctx context.Context, inDom *V1Domains.UserDomain) (outDomain V1Domains.UserDomain, err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	// Exclude soft-deleted rows — the schema keeps a deleted_at column
	// so "deleted" users remain queryable for audit/restore, but they
	// must not satisfy login or OTP flows.
	err = r.conn.GetContext(ctx, &userRecord, `SELECT * FROM users WHERE "email" = $1 AND deleted_at IS NULL`, userRecord.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return V1Domains.UserDomain{}, constants.ErrNotFound("user not found")
		}
		return V1Domains.UserDomain{}, err
	}

	return userRecord.ToV1Domain(), nil
}

func (r *postgreUserRepository) ChangeActiveUser(ctx context.Context, inDom *V1Domains.UserDomain) (err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `UPDATE users SET active = :active, updated_at = NOW() WHERE id = :id AND deleted_at IS NULL`, userRecord)

	return
}
