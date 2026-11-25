//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
)

// ByEmail returns the user with the email in whatever case it was stored, and reports an email no user has as
// not found.
func TestUserRepoByEmail(t *testing.T) {
	pool := newPool(t)
	stored := storeUser(t, pool, "ada@example.com")
	oldID := insertUserRow(t, pool, "Grace@Example.COM")
	repo := postgres.NewUserRepo(pool)
	tCases := []struct {
		name    string
		email   string
		wantID  string
		wantErr error
	}{
		{name: "a stored user is found by their email", email: "ada@example.com", wantID: stored.ID()},
		{name: "a user stored in another case is found", email: "grace@example.com", wantID: oldID},
		{name: "an email no user has is not found", email: "bob@example.com", wantErr: domain.ErrNotFound},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			address, err := domain.NewEmail(tc.email)
			if err != nil {
				t.Fatalf("NewEmail = %v", err)
			}

			got, err := repo.ByEmail(t.Context(), address)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ByEmail = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (got.ID() != tc.wantID || got.Email() != address) {
				t.Errorf("ByEmail = %s %s, want %s %s", got.ID(), got.Email(), tc.wantID, address)
			}
		})
	}
}
