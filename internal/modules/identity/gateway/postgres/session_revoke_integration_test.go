//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Revoke revokes the family of the token, its rotated successors included, and no session of another family;
// it takes a hash no session has as done.
func TestSessionRepoRevoke(t *testing.T) {
	rotateToB := func(ctx context.Context, sessions *postgres.SessionRepo) error {
		_, err := sessions.Rotate(ctx, "a", successor("b"), baseTime, grace)
		return err
	}
	tCases := []struct {
		name string
		// before is what happened first to "a", a session of the user, who also holds "other" in another family.
		before   sessionCall
		token    string
		wantLive []string
	}{
		{name: "the family of the token is revoked", token: "a", wantLive: []string{"other"}},
		{
			name:     "revoking a rotated token's family also revokes its successor",
			before:   rotateToB,
			token:    "a",
			wantLive: []string{"other"},
		},
		{name: "a token no session has is no error", token: "unknown", wantLive: []string{"a", "other"}},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := storeUser(t, pool, "ada@example.com")
			storeSession(t, pool, sessionOf(user.ID(), id.New(), "a", later))
			storeSession(t, pool, sessionOf(user.ID(), id.New(), "other", later))
			sessions := postgres.NewSessionRepo(pool)
			if tc.before != nil {
				if err := tc.before(t.Context(), sessions); err != nil {
					t.Fatalf("before = %v", err)
				}
			}

			err := sessions.Revoke(t.Context(), tc.token, baseTime)

			if err != nil {
				t.Fatalf("Revoke = %v, want nil", err)
			}
			if got := liveTokens(t, pool, baseTime); !reflect.DeepEqual(got, tc.wantLive) {
				t.Errorf("live sessions after Revoke = %v, want %v", got, tc.wantLive)
			}
		})
	}
}

// Revoke and Rotate of tokens of one family wait for one another. A rotation that holds the user when Revoke
// starts finishes first, and the session it stored is revoked with its family; a rotation that starts while
// Revoke holds the user finds its token revoked. Either way no session of the family is live afterwards.
func TestSessionRepoRevokeConcurrent(t *testing.T) {
	rotate := func(ctx context.Context, sessions *postgres.SessionRepo) error {
		_, err := sessions.Rotate(ctx, "a", successor("b"), baseTime, grace)
		return err
	}
	revoke := func(ctx context.Context, sessions *postgres.SessionRepo) error {
		return sessions.Revoke(ctx, "a", baseTime)
	}
	tCases := []struct {
		name string
		// holder runs first, in a transaction that stays open while waiter waits for it.
		holder, waiter               sessionCall
		wantHolderErr, wantWaiterErr error
	}{
		{name: "a rotation that started first is revoked with its family", holder: rotate, waiter: revoke},
		{
			name:          "a rotation that starts after finds its token revoked",
			holder:        revoke,
			waiter:        rotate,
			wantWaiterErr: domain.ErrInvalidToken,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := storeUser(t, pool, "ada@example.com")
			storeSession(t, pool, sessionOf(user.ID(), id.New(), "a", later))
			storeSession(t, pool, sessionOf(user.ID(), id.New(), "other", later))

			holderErr, waiterErr := holdThenWait(t, pool, tc.holder, tc.waiter)

			if !errors.Is(holderErr, tc.wantHolderErr) || !errors.Is(waiterErr, tc.wantWaiterErr) {
				t.Fatalf("holder, waiter = %v, %v; want %v, %v",
					holderErr, waiterErr, tc.wantHolderErr, tc.wantWaiterErr)
			}
			if got, want := liveTokens(t, pool, baseTime), []string{"other"}; !reflect.DeepEqual(got, want) {
				t.Errorf("live sessions afterwards = %v, want %v", got, want)
			}
		})
	}
}
