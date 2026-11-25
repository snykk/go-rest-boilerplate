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
)

// Use runs the check on the user's secret, one use after the other, and keeps what the check's result asks
// for: a right guess deletes it, a wrong guess saves the count of wrong guesses, and any other result, an
// expired secret and one guessed at too often among them, changes nothing. A user with no secret gets
// ErrInvalidCode.
func TestSecretRepoUse(t *testing.T) {
	const maxAttempts = 5
	errBroken := errors.New("check broken")
	guess := func(hash string, now time.Time) func(secret *domain.Secret) error {
		return func(secret *domain.Secret) error { return secret.Check(hash, now, maxAttempts) }
	}
	held := func(attempts int) *domain.Secret {
		secret := secretOf("", "code")
		secret.Attempts = attempts
		return &secret
	}
	tCases := []struct {
		name    string
		held    *domain.Secret // the user's secret before the call, without its user; nil for none
		check   func(secret *domain.Secret) error
		calls   int // uses run at the same time
		wantErr error
		want    *domain.Secret // the user's secret afterwards, without its user; nil for none
	}{
		{
			name:  "a right guess deletes the secret",
			held:  held(1),
			check: guess("code", baseTime),
			calls: 1,
		},
		{
			name:    "a wrong guess saves the attempt",
			held:    held(1),
			check:   guess("wrong", baseTime),
			calls:   1,
			wantErr: domain.ErrInvalidCode,
			want:    held(2),
		},
		{
			name:    "concurrent wrong guesses all count",
			held:    held(0),
			check:   guess("wrong", baseTime),
			calls:   4,
			wantErr: domain.ErrInvalidCode,
			want:    held(4),
		},
		{
			name:    "an expired secret is kept as it was",
			held:    held(1),
			check:   guess("code", later.Add(time.Second)),
			calls:   1,
			wantErr: domain.ErrCodeExpired,
			want:    held(1),
		},
		{
			name:    "a secret guessed at too often is kept as it was",
			held:    held(maxAttempts),
			check:   guess("code", baseTime),
			calls:   1,
			wantErr: domain.ErrTooManyAttempts,
			want:    held(maxAttempts),
		},
		{
			name: "a check failing otherwise changes nothing",
			held: held(1),
			check: func(secret *domain.Secret) error {
				secret.Attempts = 9
				return errBroken
			},
			calls:   1,
			wantErr: errBroken,
			want:    held(1),
		},
		{
			name:    "a user with no secret gets ErrInvalidCode",
			check:   guess("code", baseTime),
			calls:   1,
			wantErr: domain.ErrInvalidCode,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := storeUser(t, pool, "ada@example.com")
			repo := postgres.NewVerificationRepo(pool)
			if tc.held != nil {
				secret := *tc.held
				secret.UserID = user.ID()
				put(t, repo, secret)
			}
			failures := make([]error, tc.calls)
			var running sync.WaitGroup

			for i := range tc.calls {
				running.Go(func() { failures[i] = repo.Use(t.Context(), user.ID(), tc.check) })
			}
			running.Wait()

			for _, err := range failures {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Use = %v, want %v", err, tc.wantErr)
				}
			}
			got, err := repo.ByHash(t.Context(), "code")
			if tc.want == nil {
				if !errors.Is(err, domain.ErrInvalidToken) {
					t.Errorf("ByHash after Use = %+v, %v; want no secret", got, err)
				}
				return
			}
			want := *tc.want
			want.UserID = user.ID()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("ByHash after Use = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

// Use keeps a secret it refuses as expired or as guessed at too often: the right code stays refused, and the
// secret stays the user's latest, so a Put within the cooldown of its creation stores nothing.
func TestSecretRepoUseEnded(t *testing.T) {
	const maxAttempts = 5
	tCases := []struct {
		name     string
		attempts int       // the wrong guesses made on the held secret
		at       time.Time // when the right code is presented
		wantErr  error
	}{
		{name: "a secret guessed at too often", attempts: maxAttempts, at: baseTime, wantErr: domain.ErrTooManyAttempts},
		{name: "an expired secret", at: later.Add(time.Second), wantErr: domain.ErrCodeExpired},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := storeUser(t, pool, "ada@example.com")
			repo := postgres.NewVerificationRepo(pool)
			held := secretOf(user.ID(), "code")
			held.Attempts = tc.attempts
			put(t, repo, held)
			rightCode := func(secret *domain.Secret) error { return secret.Check("code", tc.at, maxAttempts) }

			for attempt := range 2 {
				if err := repo.Use(t.Context(), user.ID(), rightCode); !errors.Is(err, tc.wantErr) {
					t.Fatalf("Use with the right code, attempt %d = %v, want %v", attempt+1, err, tc.wantErr)
				}
			}
			next := secretOf(user.ID(), "next")
			next.CreatedAt = held.CreatedAt.Add(cooldown - time.Second)
			stored, err := repo.Put(t.Context(), next, cooldown)

			if err != nil || stored {
				t.Errorf("Put within the cooldown = %t, %v; want false, nil", stored, err)
			}
			if got, err := repo.ByHash(t.Context(), "code"); err != nil || !reflect.DeepEqual(got, held) {
				t.Errorf("ByHash(code) = %+v, %v; want %+v", got, err, held)
			}
		})
	}
}
