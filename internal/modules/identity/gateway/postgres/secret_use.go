package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Use loads the user's secret of the repository's kind and runs check on it while it holds the secret's row
// lock, so concurrent uses of one secret apply one after the other. It deletes the secret when check returns
// nil, saves the secret's Attempts when check returns domain.ErrInvalidCode, and changes nothing when check
// returns another error. A secret refused as domain.ErrCodeExpired or domain.ErrTooManyAttempts therefore
// stays, refused by every later check, with its CreatedAt, from which Put counts the cooldown. It returns
// check's error, and domain.ErrInvalidCode when the user holds no secret of the kind, including when userID is
// not a UUID. It runs in a transaction of its own, or in a savepoint when the repository runs on a transaction.
func (r *SecretRepo) Use(ctx context.Context, userID string, check func(secret *domain.Secret) error) error {
	var checked error
	err := pgx.BeginFunc(ctx, r.db, func(transaction pgx.Tx) error {
		secret, err := r.lock(ctx, transaction, userID)
		if err != nil {
			return err
		}

		checked = check(&secret)
		return r.settle(ctx, transaction, userID, secret.Attempts, checked)
	})
	if err != nil {
		return err
	}
	return checked
}

func (r *SecretRepo) lock(ctx context.Context, transaction pgx.Tx, userID string) (domain.Secret, error) {
	rows, err := transaction.Query(ctx,
		`SELECT `+kindColumns+` FROM `+r.table+` WHERE user_id = $1 FOR UPDATE`, userID)
	if err != nil {
		return domain.Secret{}, translateSecret(err, "lock secret")
	}
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[secretRow])
	if err != nil {
		return domain.Secret{}, translateSecret(err, "lock secret")
	}
	return row.toDomain(), nil
}

// settle stores what checked, the result of a check of the user's secret, asks for: no secret after a match, or
// the secret with its count of wrong guesses at attempts after a wrong guess. For any other result it returns
// checked, so nothing changes.
func (r *SecretRepo) settle(
	ctx context.Context, transaction pgx.Tx, userID string, attempts int, checked error,
) error {
	switch {
	case checked == nil:
		if _, err := transaction.Exec(ctx, `DELETE FROM `+r.table+` WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("delete secret: %w", err)
		}
		return nil
	case errors.Is(checked, domain.ErrInvalidCode):
		_, err := transaction.Exec(ctx, `UPDATE `+r.table+` SET attempts = $2 WHERE user_id = $1`, userID, attempts)
		if err != nil {
			return fmt.Errorf("count the wrong guess: %w", err)
		}
		return nil
	}
	return checked
}

func translateSecret(err error, operation string) error {
	if errors.Is(err, pgx.ErrNoRows) || malformedID(err) {
		return domain.ErrInvalidCode
	}
	return fmt.Errorf("%s: %w", operation, err)
}
