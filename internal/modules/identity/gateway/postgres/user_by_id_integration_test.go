//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// ByID returns a stored user as it was stored, and reports an unknown or malformed ID as not found.
func TestUserRepoByID(t *testing.T) {
	pool := newPool(t)
	stored := newUser(t, "ada@example.com")
	stored.Verify(baseTime)
	stored.FailLogin(baseTime, 1, time.Minute)
	repo := postgres.NewUserRepo(pool)
	if err := repo.Create(t.Context(), stored); err != nil {
		t.Fatalf("Create = %v", err)
	}
	tCases := []struct {
		name    string
		userID  string
		want    *domain.User
		wantErr error
	}{
		{name: "a stored user is returned as it was stored", userID: stored.ID(), want: stored},
		{name: "an unknown ID is not found", userID: id.New(), wantErr: domain.ErrNotFound},
		{name: "a malformed ID is not found", userID: "not-a-uuid", wantErr: domain.ErrNotFound},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.ByID(t.Context(), tc.userID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ByID = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ByID = %+v, want %+v", got, tc.want)
			}
		})
	}
}
