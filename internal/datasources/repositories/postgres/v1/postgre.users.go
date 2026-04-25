package v1

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
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

func (r *postgreUserRepository) Store(ctx context.Context, inDom *V1Domains.UserDomain) (V1Domains.UserDomain, error) {
	userRecord := records.FromUsersV1Domain(inDom)

	// INSERT ... RETURNING * so the caller gets the persisted row in
	// one round-trip. Previously we did INSERT then GetByEmail; if
	// GetByEmail failed (network blip, replica lag) the INSERT was
	// already committed and the user was orphaned in the response.
	rows, err := r.conn.NamedQueryContext(ctx, `
		INSERT INTO users(id, username, email, password, active, role_id, created_at)
		VALUES (uuid_generate_v4(), :username, :email, :password, false, :role_id, :created_at)
		RETURNING id, username, email, password, active, role_id, created_at, updated_at, deleted_at
	`, userRecord)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return V1Domains.UserDomain{}, apperror.Conflict("username or email already exists")
		}
		return V1Domains.UserDomain{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		// RETURNING never produces zero rows on a successful INSERT,
		// but check anyway so a future schema change can't silently
		// hand back an empty struct.
		return V1Domains.UserDomain{}, fmt.Errorf("insert succeeded but RETURNING produced no row")
	}
	var stored records.Users
	if err := rows.StructScan(&stored); err != nil {
		return V1Domains.UserDomain{}, fmt.Errorf("scan inserted user: %w", err)
	}
	return stored.ToV1Domain(), nil
}

func (r *postgreUserRepository) GetByEmail(ctx context.Context, inDom *V1Domains.UserDomain) (outDomain V1Domains.UserDomain, err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	// Exclude soft-deleted rows — the schema keeps a deleted_at column
	// so "deleted" users remain queryable for audit/restore, but they
	// must not satisfy login or OTP flows.
	err = r.conn.GetContext(ctx, &userRecord, `SELECT * FROM users WHERE "email" = $1 AND deleted_at IS NULL`, userRecord.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return V1Domains.UserDomain{}, apperror.NotFound("user not found")
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
