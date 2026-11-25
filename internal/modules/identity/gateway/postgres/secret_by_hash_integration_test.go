//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
)

// ByHash returns the one secret with the hash, and refuses a hash no secret has or that several users'
// secrets share, as a six-digit code can be.
func TestSecretRepoByHash(t *testing.T) {
	pool := newPool(t)
	ada := storeUser(t, pool, "ada@example.com")
	bob := storeUser(t, pool, "bob@example.com")
	carol := storeUser(t, pool, "carol@example.com")
	repo := postgres.NewVerificationRepo(pool)
	own := secretOf(ada.ID(), "own")
	put(t, repo, own)
	put(t, repo, secretOf(bob.ID(), "shared"))
	put(t, repo, secretOf(carol.ID(), "shared"))
	tCases := []struct {
		name    string
		hash    string
		want    domain.Secret
		wantErr error
	}{
		{name: "a stored secret is found by its hash", hash: "own", want: own},
		{name: "a hash no secret has is ErrInvalidToken", hash: "unknown", wantErr: domain.ErrInvalidToken},
		{name: "a hash two users hold is ErrInvalidToken", hash: "shared", wantErr: domain.ErrInvalidToken},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.ByHash(t.Context(), tc.hash)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ByHash = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ByHash = %+v, want %+v", got, tc.want)
			}
		})
	}
}
