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

// RevokeAll revokes every live session of the user, in every family, and no session of another user.
func TestSessionRepoRevokeAll(t *testing.T) {
	tCases := []struct {
		name     string
		user     func(userID string) string
		wantLive []string
	}{
		{
			name:     "every live session of the user is revoked",
			user:     func(userID string) string { return userID },
			wantLive: []string{"bob"},
		},
		{
			name:     "a user with no session is no error",
			user:     func(string) string { return id.New() },
			wantLive: []string{"a", "b", "bob"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			ada := storeUser(t, pool, "ada@example.com")
			bob := storeUser(t, pool, "bob@example.com")
			storeSession(t, pool, sessionOf(ada.ID(), id.New(), "a", later))
			storeSession(t, pool, sessionOf(ada.ID(), id.New(), "b", later))
			storeSession(t, pool, sessionOf(bob.ID(), id.New(), "bob", later))

			err := postgres.NewSessionRepo(pool).RevokeAll(t.Context(), tc.user(ada.ID()), baseTime)

			if err != nil {
				t.Fatalf("RevokeAll = %v, want nil", err)
			}
			if got := liveTokens(t, pool, baseTime); !reflect.DeepEqual(got, tc.wantLive) {
				t.Errorf("live sessions after RevokeAll = %v, want %v", got, tc.wantLive)
			}
		})
	}
}

// RevokeAll and Rotate for the same user wait for one another. A rotation that holds the user when RevokeAll
// starts finishes first, and the session it stored is revoked too; a rotation that starts while RevokeAll
// holds the user finds its token revoked. Either way no session of the user is live afterwards.
func TestSessionRepoRevokeAllConcurrent(t *testing.T) {
	rotate := func(ctx context.Context, sessions *postgres.SessionRepo, _ string) error {
		_, err := sessions.Rotate(ctx, "a", successor("b"), baseTime, grace)
		return err
	}
	revokeAll := func(ctx context.Context, sessions *postgres.SessionRepo, userID string) error {
		return sessions.RevokeAll(ctx, userID, baseTime)
	}
	tCases := []struct {
		name string
		// holder runs first, in a transaction that stays open while waiter waits for it.
		holder, waiter               func(ctx context.Context, sessions *postgres.SessionRepo, userID string) error
		wantHolderErr, wantWaiterErr error
	}{
		{name: "a rotation that started first is revoked with the rest", holder: rotate, waiter: revokeAll},
		{
			name:          "a rotation that starts after finds its token revoked",
			holder:        revokeAll,
			waiter:        rotate,
			wantWaiterErr: domain.ErrInvalidToken,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			ada := storeUser(t, pool, "ada@example.com")
			bob := storeUser(t, pool, "bob@example.com")
			storeSession(t, pool, sessionOf(ada.ID(), id.New(), "a", later))
			storeSession(t, pool, sessionOf(bob.ID(), id.New(), "bob", later))
			forAda := func(call func(context.Context, *postgres.SessionRepo, string) error) sessionCall {
				return func(ctx context.Context, sessions *postgres.SessionRepo) error {
					return call(ctx, sessions, ada.ID())
				}
			}

			holderErr, waiterErr := holdThenWait(t, pool, forAda(tc.holder), forAda(tc.waiter))

			if !errors.Is(holderErr, tc.wantHolderErr) || !errors.Is(waiterErr, tc.wantWaiterErr) {
				t.Fatalf("holder, waiter = %v, %v; want %v, %v",
					holderErr, waiterErr, tc.wantHolderErr, tc.wantWaiterErr)
			}
			if got, want := liveTokens(t, pool, baseTime), []string{"bob"}; !reflect.DeepEqual(got, want) {
				t.Errorf("live sessions afterwards = %v, want %v", got, want)
			}
		})
	}
}
