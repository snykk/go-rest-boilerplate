//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
)

// Create stores a new user as it is, and refuses an email another user holds in any case.
func TestUserRepoCreate(t *testing.T) {
	tCases := []struct {
		name     string
		existing string // the email of a user stored first, exactly as written; "" for none
		wantErr  error
	}{
		{name: "a new user is stored as it is"},
		{
			name:     "an email another user has is ErrEmailTaken",
			existing: "ada@example.com",
			wantErr:  domain.ErrEmailTaken,
		},
		{
			name:     "an email taken in another case is ErrEmailTaken",
			existing: "Ada@Example.COM",
			wantErr:  domain.ErrEmailTaken,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			if tc.existing != "" {
				insertUserRow(t, pool, tc.existing)
			}
			repo := postgres.NewUserRepo(pool)
			user := newUser(t, "ada@example.com")

			err := repo.Create(t.Context(), user)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Create = %v, want %v", err, tc.wantErr)
			}
			got, err := repo.ByID(t.Context(), user.ID())
			if stored := err == nil; stored != (tc.wantErr == nil) {
				t.Fatalf("ByID after Create = %v, want stored: %t", err, tc.wantErr == nil)
			}
			if tc.wantErr == nil && !reflect.DeepEqual(got, user) {
				t.Errorf("ByID after Create = %+v, want %+v", got, user)
			}
		})
	}
}
