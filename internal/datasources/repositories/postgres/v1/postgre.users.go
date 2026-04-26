package v1

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

type postgreUserRepository struct {
	conn *sqlx.DB
}

func NewUserRepository(conn *sqlx.DB) usecases.UserRepository {
	return &postgreUserRepository{
		conn: conn,
	}
}

func (r *postgreUserRepository) Store(ctx context.Context, inDom *entities.UserDomain) (entities.UserDomain, error) {
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
			return entities.UserDomain{}, apperror.Conflict("username or email already exists")
		}
		return entities.UserDomain{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		// RETURNING never produces zero rows on a successful INSERT,
		// but check anyway so a future schema change can't silently
		// hand back an empty struct.
		return entities.UserDomain{}, fmt.Errorf("insert succeeded but RETURNING produced no row")
	}
	var stored records.Users
	if err := rows.StructScan(&stored); err != nil {
		return entities.UserDomain{}, fmt.Errorf("scan inserted user: %w", err)
	}
	return stored.ToV1Domain(), nil
}

func (r *postgreUserRepository) GetByEmail(ctx context.Context, inDom *entities.UserDomain) (outDomain entities.UserDomain, err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	// Exclude soft-deleted rows — the schema keeps a deleted_at column
	// so "deleted" users remain queryable for audit/restore, but they
	// must not satisfy login or OTP flows.
	err = r.conn.GetContext(ctx, &userRecord, `SELECT * FROM users WHERE "email" = $1 AND deleted_at IS NULL`, userRecord.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.UserDomain{}, apperror.NotFound("user not found")
		}
		return entities.UserDomain{}, err
	}

	return userRecord.ToV1Domain(), nil
}

func (r *postgreUserRepository) ChangeActiveUser(ctx context.Context, inDom *entities.UserDomain) (err error) {
	userRecord := records.FromUsersV1Domain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `UPDATE users SET active = :active, updated_at = NOW() WHERE id = :id AND deleted_at IS NULL`, userRecord)

	return
}

func (r *postgreUserRepository) GetByID(ctx context.Context, id string) (entities.UserDomain, error) {
	var stored records.Users
	err := r.conn.GetContext(ctx, &stored, `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.UserDomain{}, apperror.NotFound("user not found")
		}
		return entities.UserDomain{}, err
	}
	return stored.ToV1Domain(), nil
}

// hardLimit caps List page size so a misbehaving caller can't pull
// the whole table in one request. Repeating the cap here (in addition
// to whatever clamping the handler does) is defense in depth.
const hardLimit = 200

func (r *postgreUserRepository) List(ctx context.Context, filter usecases.ListFilter, offset, limit int) ([]entities.UserDomain, error) {
	if limit <= 0 || limit > hardLimit {
		limit = hardLimit
	}
	if offset < 0 {
		offset = 0
	}

	// Build the WHERE clause dynamically. Each predicate appends an
	// $N placeholder + value to keep the query parameterized — never
	// concatenate filter values into the SQL string.
	var (
		where = []string{}
		args  = []interface{}{}
		idx   = 1
	)
	if !filter.IncludeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if filter.RoleID != 0 {
		where = append(where, fmt.Sprintf("role_id = $%d", idx))
		args = append(args, filter.RoleID)
		idx++
	}
	if filter.ActiveOnly {
		where = append(where, "active = true")
	}

	query := "SELECT * FROM users"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	var rows []records.Users
	if err := r.conn.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	out := make([]entities.UserDomain, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToV1Domain())
	}
	return out, nil
}

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
