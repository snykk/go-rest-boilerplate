package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// secretFixture is a secret of user-1 whose hash is "abc", with the given expiry and wrong guesses so far.
func secretFixture(expiresAt time.Time, attempts int) domain.Secret {
	return domain.Secret{
		UserID:    "user-1",
		Hash:      "abc",
		ExpiresAt: expiresAt,
		Attempts:  attempts,
	}
}

// Check accepts the hash that matches, counts a wrong hash as an attempt, and refuses an expired secret and
// one that used all its attempts, even with the right hash.
func TestSecretCheck(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	valid := now.Add(time.Minute)
	const maxAttempts = 3
	tCases := []struct {
		name         string
		secret       domain.Secret
		hash         string
		wantErr      error
		wantAttempts int
	}{
		{name: "the right hash matches", secret: secretFixture(valid, 0), hash: "abc"},
		{
			name:         "the right hash matches at the last allowed attempt",
			secret:       secretFixture(valid, maxAttempts-1),
			hash:         "abc",
			wantAttempts: maxAttempts - 1,
		},
		{name: "the right hash matches at the moment it expires", secret: secretFixture(now, 0), hash: "abc"},
		{
			name:         "a wrong hash counts an attempt",
			secret:       secretFixture(valid, 0),
			hash:         "abd",
			wantErr:      domain.ErrInvalidCode,
			wantAttempts: 1,
		},
		{
			name:         "a hash of another length counts an attempt",
			secret:       secretFixture(valid, 0),
			hash:         "abcd",
			wantErr:      domain.ErrInvalidCode,
			wantAttempts: 1,
		},
		{
			name:         "a wrong hash adds to the attempts already made",
			secret:       secretFixture(valid, 1),
			hash:         "abd",
			wantErr:      domain.ErrInvalidCode,
			wantAttempts: 2,
		},
		{
			name:    "an expired secret is refused, even with the right hash",
			secret:  secretFixture(now.Add(-time.Nanosecond), 0),
			hash:    "abc",
			wantErr: domain.ErrCodeExpired,
		},
		{
			name:    "an expired secret does not count an attempt",
			secret:  secretFixture(now.Add(-time.Minute), 0),
			hash:    "abd",
			wantErr: domain.ErrCodeExpired,
		},
		{
			name:         "too many attempts are refused, even with the right hash",
			secret:       secretFixture(valid, maxAttempts),
			hash:         "abc",
			wantErr:      domain.ErrTooManyAttempts,
			wantAttempts: maxAttempts,
		},
		{
			name:         "too many attempts do not count another",
			secret:       secretFixture(valid, maxAttempts+2),
			hash:         "abd",
			wantErr:      domain.ErrTooManyAttempts,
			wantAttempts: maxAttempts + 2,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			secret := tc.secret

			err := secret.Check(tc.hash, now, maxAttempts)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Check = %v, want %v", err, tc.wantErr)
			}
			if secret.Attempts != tc.wantAttempts {
				t.Errorf("Check left %d attempts, want %d", secret.Attempts, tc.wantAttempts)
			}
		})
	}
}

// Check lets wrong hashes use up the attempts one by one, and then refuses even the right hash.
func TestSecretCheckAttemptsRunOut(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	secret := secretFixture(now.Add(time.Minute), 0)

	for i := range 2 {
		if err := secret.Check("wrong", now, 2); !errors.Is(err, domain.ErrInvalidCode) {
			t.Fatalf("Check #%d with a wrong hash = %v, want %v", i+1, err, domain.ErrInvalidCode)
		}
	}

	if err := secret.Check("abc", now, 2); !errors.Is(err, domain.ErrTooManyAttempts) {
		t.Errorf("Check with the right hash after two wrong ones = %v, want %v", err, domain.ErrTooManyAttempts)
	}
}
