package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/go-rest-boilerplate/internal/apperror"
	"github.com/go-rest-boilerplate/internal/business/domain"
	"github.com/go-rest-boilerplate/internal/datasources/records"
	"github.com/go-rest-boilerplate/pkg/logger"
)

func (r *postgresPlayersRepository) Store(ctx context.Context, inDom *domain.Player) (domain.Player, error) {
	const (
		repositoryName = "players"
		funcName       = "Store"
		queryName      = "insertPlayer"
		fileName       = "players.store.go"
	)
	rec := records.FromPlayersV1Domain(inDom)

	var stored records.Player
	err := r.conn.GetContext(ctx, &stored, `
		INSERT INTO players(id, installation_id, role_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, installation_id, role_id, created_at
	`, rec.Id, rec.InstallationId, rec.RoleId, rec.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			logger.ErrorWithContext(ctx, "Failed to insert player: unique violation", logger.Fields{
				"repository": repositoryName,
				"method":     funcName,
				"query":      queryName,
				"file":       fileName,
				"error":      err.Error(),
				"table":      "players",
			})
			return domain.Player{}, apperror.Conflict("installation id already registered")
		}
		logger.ErrorWithContext(ctx, "Failed to insert player into database", logger.Fields{
			"repository": repositoryName,
			"method":     funcName,
			"query":      queryName,
			"file":       fileName,
			"error":      err.Error(),
			"table":      "players",
		})
		return domain.Player{}, fmt.Errorf("insert player: %w", err)
	}
	return stored.ToV1Domain(), nil
}