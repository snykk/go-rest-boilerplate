//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
)

// cooldown is the cooldown the tests put secrets with.
const cooldown = time.Minute

// Put stores a secret, with the time it was created, as its user's only secret of the repository's kind. It
// replaces the secret the user held once that one is the cooldown old, keeps a younger one and stores nothing
// instead, and leaves the secrets of the other kind alone.
func TestSecretRepoPut(t *testing.T) {
	tCases := []struct {
		name string
		// heldAge is how old the secret "held" the user holds, with two wrong guesses, is when the new secret is
		// created; the user then holds a reset token "reset" too. Zero means the user holds no secret.
		heldAge    time.Duration
		wantStored bool
	}{
		{name: "a first secret is stored with the time it was created", wantStored: true},
		{name: "a new secret replaces one held for the cooldown", heldAge: cooldown, wantStored: true},
		{
			name:    "a new secret within the cooldown is not stored, and the held one is kept",
			heldAge: cooldown - time.Second,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := storeUser(t, pool, "ada@example.com")
			verifications, resets := postgres.NewVerificationRepo(pool), postgres.NewResetRepo(pool)
			held := secretOf(user.ID(), "held")
			held.Attempts, held.CreatedAt = 2, baseTime.Add(-tc.heldAge)
			if tc.heldAge > 0 {
				put(t, verifications, held)
				put(t, resets, secretOf(user.ID(), "reset"))
			}
			secret := secretOf(user.ID(), "new")

			stored, err := verifications.Put(t.Context(), secret, cooldown)

			if err != nil || stored != tc.wantStored {
				t.Fatalf("Put = %t, %v; want %t, nil", stored, err, tc.wantStored)
			}
			kept, gone := held, secret
			if tc.wantStored {
				kept, gone = secret, held
			}
			if got, err := verifications.ByHash(t.Context(), kept.Hash); err != nil || !reflect.DeepEqual(got, kept) {
				t.Errorf("ByHash(%s) = %+v, %v; want %+v", kept.Hash, got, err, kept)
			}
			if _, err := verifications.ByHash(t.Context(), gone.Hash); !errors.Is(err, domain.ErrInvalidToken) {
				t.Errorf("ByHash(%s) = %v, want %v", gone.Hash, err, domain.ErrInvalidToken)
			}
			if _, err := resets.ByHash(t.Context(), "new"); !errors.Is(err, domain.ErrInvalidToken) {
				t.Errorf("resets.ByHash(new) = %v, want %v: the kinds met", err, domain.ErrInvalidToken)
			}
			if _, err := resets.ByHash(t.Context(), "reset"); (err == nil) != (tc.heldAge > 0) {
				t.Errorf("resets.ByHash(reset) = %v, want found: %t", err, tc.heldAge > 0)
			}
		})
	}
}

// Put applies concurrent puts for one user one after the other: a put that waits for another's transaction
// finds the secret that one stored, and keeps it within the cooldown.
func TestSecretRepoPutConcurrent(t *testing.T) {
	pool := newPool(t)
	user := storeUser(t, pool, "ada@example.com")
	transaction, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("Begin = %v", err)
	}
	defer func() { _ = transaction.Rollback(context.Background()) }()
	first, second := secretOf(user.ID(), "first"), secretOf(user.ID(), "second")
	stored, err := postgres.NewVerificationRepo(transaction).Put(t.Context(), first, cooldown)
	if !stored || err != nil {
		t.Fatalf("Put(first) = %t, %v; want true, nil", stored, err)
	}

	type result struct {
		stored bool
		err    error
	}
	waited := make(chan result, 1)
	go func() {
		stored, err := postgres.NewVerificationRepo(pool).Put(t.Context(), second, cooldown)
		waited <- result{stored, err}
	}()
	waitForLockWaiter(t, pool)
	if err := transaction.Commit(t.Context()); err != nil {
		t.Fatalf("Commit = %v", err)
	}

	if got := <-waited; got.stored || got.err != nil {
		t.Errorf("Put(second) = %t, %v; want false, nil", got.stored, got.err)
	}
	if got, err := postgres.NewVerificationRepo(pool).ByHash(t.Context(), "first"); err != nil ||
		!reflect.DeepEqual(got, first) {
		t.Errorf("ByHash(first) = %+v, %v; want %+v", got, err, first)
	}
}
