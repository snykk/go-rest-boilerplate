package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

func (r *postgreUserRepository) Store(ctx context.Context, inDom *domain.User) (domain.User, error) {
	const (
		repositoryName = "users"
		funcName       = "Store"
		queryName      = "insertUser"
		fileName       = "users.store.go"
	)
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
			logger.ErrorWithContext(ctx, "Failed to insert user: unique violation", logger.Fields{
				"repository": repositoryName,
				"method":     funcName,
				"query":      queryName,
				"file":       fileName,
				"error":      err.Error(),
				"table":      "users",
				"email":      userRecord.Email,
			})
			return domain.User{}, apperror.Conflict("username or email already exists")
		}
		logger.ErrorWithContext(ctx, "Failed to insert user into database", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
		})
		return domain.User{}, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		// RETURNING never produces zero rows on a successful INSERT,
		// but check anyway so a future schema change can't silently
		// hand back an empty struct.
		err := fmt.Errorf("insert succeeded but RETURNING produced no row")
		logger.ErrorWithContext(ctx, "Insert returned no row", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
		})
		return domain.User{}, err
	}
	var stored records.Users
	if err := rows.StructScan(&stored); err != nil {
		logger.ErrorWithContext(ctx, "Failed to scan inserted user row", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "users",
		})
		return domain.User{}, fmt.Errorf("scan inserted user: %w", err)
	}
	return stored.ToV1Domain(), nil
}
