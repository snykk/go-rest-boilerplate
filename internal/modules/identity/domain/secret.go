package domain

import (
	"crypto/subtle"
	"time"
)

// Secret is a short-lived value a user must present once, as a store keeps it: an e-mail verification code
// or a password reset token. Only its SHA-256 hash is kept, with a limit on wrong guesses.
type Secret struct {
	UserID    string
	Hash      string
	ExpiresAt time.Time
	// Attempts is the number of wrong guesses made so far.
	Attempts int
	// CreatedAt is when the secret was issued.
	CreatedAt time.Time
}

// Check compares hash with the secret's hash in constant time. It returns ErrCodeExpired when now is after
// ExpiresAt, ErrTooManyAttempts when Attempts has reached maxAttempts, even if hash is right, and
// ErrInvalidCode for a wrong hash, after adding one to Attempts. It returns nil on a match, which leaves the
// secret as it is.
func (s *Secret) Check(hash string, now time.Time, maxAttempts int) error {
	if now.After(s.ExpiresAt) {
		return ErrCodeExpired
	}
	if s.Attempts >= maxAttempts {
		return ErrTooManyAttempts
	}
	if subtle.ConstantTimeCompare([]byte(s.Hash), []byte(hash)) != 1 {
		s.Attempts++
		return ErrInvalidCode
	}
	return nil
}
