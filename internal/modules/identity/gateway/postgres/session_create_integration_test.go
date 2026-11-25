//go:build integration

package postgres_test

import (
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Create stores a live session of a stored user, and refuses a session of a user the store does not hold.
func TestSessionRepoCreate(t *testing.T) {
	tCases := []struct {
		name       string
		storedUser bool
		wantErr    bool
		wantLive   []string
	}{
		{name: "a session of a stored user is stored live", storedUser: true, wantLive: []string{"a"}},
		{name: "a session of an unknown user is refused", wantErr: true, wantLive: []string{}},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			userID := id.New()
			if tc.storedUser {
				userID = storeUser(t, pool, "ada@example.com").ID()
			}

			err := postgres.NewSessionRepo(pool).Create(t.Context(), sessionOf(userID, id.New(), "a", later))

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("Create = %v, want error: %t", err, tc.wantErr)
			}
			if got := liveTokens(t, pool, baseTime); !reflect.DeepEqual(got, tc.wantLive) {
				t.Errorf("live sessions after Create = %v, want %v", got, tc.wantLive)
			}
		})
	}
}
