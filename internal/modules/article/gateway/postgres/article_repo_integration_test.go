//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

// NewArticleRepo works on a pool or on a transaction. On a transaction every call, the row-locked Update
// included, is kept when the transaction commits and undone when it rolls back.
func TestNewArticleRepo(t *testing.T) {
	errBoom := errors.New("boom")
	tCases := []struct {
		name       string
		fail       bool // the transaction returns errBoom after its calls, so it rolls back
		wantErr    error
		wantStored bool
	}{
		{name: "calls on a transaction that commits are all kept", wantStored: true},
		{name: "calls on a transaction that rolls back are all undone", fail: true, wantErr: errBoom},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := testenv.Postgres(t, migrations.FS)
			article := draft(t, "a", baseTime)

			err := pgx.BeginFunc(t.Context(), pool, func(transaction pgx.Tx) error {
				repo := postgres.NewArticleRepo(transaction)
				if err := repo.Create(t.Context(), article); err != nil {
					return err
				}
				if _, err := repo.Update(t.Context(), article.ID(), func(article *domain.Article) error {
					return article.Publish(baseTime)
				}); err != nil {
					return err
				}
				if tc.fail {
					return errBoom
				}
				return nil
			})

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("BeginFunc = %v, want %v", err, tc.wantErr)
			}
			got, err := postgres.NewArticleRepo(pool).ByID(t.Context(), article.ID())
			if stored := err == nil; stored != tc.wantStored {
				t.Fatalf("stored = %t (%v), want %t", stored, err, tc.wantStored)
			}
			if tc.wantStored && got.Status() != domain.StatusPublished {
				t.Errorf("status = %s, want published: the update on the transaction was lost", got.Status())
			}
		})
	}
}
