package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ByHash returns the secret of the repository's kind whose hash is the given one. It returns
// domain.ErrInvalidToken when no secret has that hash, and when several do, as the hash of a verification code
// several users hold can: such a hash names none of them.
func (r *SecretRepo) ByHash(ctx context.Context, hash string) (domain.Secret, error) {
	rows, err := r.db.Query(ctx, `SELECT `+kindColumns+` FROM `+r.table+` WHERE hash = $1 LIMIT 2`, hash)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("select secret: %w", err)
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[secretRow])
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, pgx.ErrTooManyRows) {
		return domain.Secret{}, domain.ErrInvalidToken
	}
	if err != nil {
		return domain.Secret{}, fmt.Errorf("select secret: %w", err)
	}
	return row.toDomain(), nil
}
