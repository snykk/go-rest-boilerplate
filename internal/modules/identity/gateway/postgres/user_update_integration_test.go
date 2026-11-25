//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Update saves every field of the change it applies to a stored user and returns the result, applying
// concurrent changes one after the other. It saves nothing when the change fails, and reports an unknown or
// malformed ID as not found.
func TestUserRepoUpdate(t *testing.T) {
	changedAt := baseTime.Add(time.Minute)
	errRefused := errors.New("change refused")
	storedID := func(user *domain.User) string { return user.ID() }
	failLogin := func(user *domain.User) error {
		user.FailLogin(changedAt, 100, time.Hour)
		return nil
	}
	tCases := []struct {
		name      string
		userID    func(stored *domain.User) string
		change    func(user *domain.User) error
		calls     int // updates run at the same time
		wantErr   error
		wantSaved bool // the stored user went through change calls times
	}{
		{
			name:   "a change to a stored user is saved and returned",
			userID: storedID,
			change: func(user *domain.User) error {
				user.Verify(changedAt)
				user.SetPasswordHash("new-hash", changedAt)
				return nil
			},
			calls:     1,
			wantSaved: true,
		},
		{
			name:   "a change that fails saves nothing",
			userID: storedID,
			change: func(user *domain.User) error {
				user.Verify(changedAt)
				return errRefused
			},
			calls:   1,
			wantErr: errRefused,
		},
		{
			name:      "concurrent failed logins all count",
			userID:    storedID,
			change:    failLogin,
			calls:     10,
			wantSaved: true,
		},
		{
			name:    "an unknown ID is not found",
			userID:  func(*domain.User) string { return id.New() },
			change:  failLogin,
			calls:   1,
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "a malformed ID is not found",
			userID:  func(*domain.User) string { return "not-a-uuid" },
			change:  failLogin,
			calls:   1,
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			stored := storeUser(t, pool, "ada@example.com")
			repo := postgres.NewUserRepo(pool)
			results := make([]*domain.User, tc.calls)
			failures := make([]error, tc.calls)
			var running sync.WaitGroup

			for i := range tc.calls {
				running.Go(func() { results[i], failures[i] = repo.Update(t.Context(), tc.userID(stored), tc.change) })
			}
			running.Wait()

			for _, err := range failures {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Update = %v, want %v", err, tc.wantErr)
				}
			}
			want := domain.Restore(stored.Stored())
			if tc.wantSaved {
				for range tc.calls {
					_ = tc.change(want)
				}
			}
			read, err := repo.ByID(t.Context(), stored.ID())
			if err != nil || !reflect.DeepEqual(read, want) {
				t.Fatalf("ByID after Update = %+v, %v; want %+v", read, err, want)
			}
			if tc.calls == 1 && tc.wantErr == nil && !reflect.DeepEqual(results[0], read) {
				t.Errorf("Update = %+v, want what is stored: %+v", results[0], read)
			}
		})
	}
}
