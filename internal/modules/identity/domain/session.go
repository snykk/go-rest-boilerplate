package domain

import "time"

// Session is one refresh token of a user, as a store keeps it. Only the SHA-256 hash of the token is kept.
// Every token issued by one sign-in, and by the rotations that follow it, shares a Family.
type Session struct {
	ID        string
	UserID    string
	Family    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
