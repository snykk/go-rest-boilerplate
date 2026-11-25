package domain

import "errors"

// Errors of the identity rules and of the identity store. Their messages are written for API clients and are
// safe to return to them.
var (
	ErrNotFound           = errors.New("user not found")
	ErrEmailTaken         = errors.New("email is already used by another account")
	ErrInvalidEmail       = errors.New("email must be one address of at most 254 bytes with no quote, bracket or space")
	ErrWeakPassword       = errors.New("password must be 8 to 72 bytes")
	ErrInvalidDisplayName = errors.New("display name must be 1-50 characters, none of them a control character")
	ErrInvalidCredentials = errors.New("email or password is wrong")
	ErrEmailNotVerified   = errors.New("email is not verified")
	ErrAccountLocked      = errors.New("account is locked after too many failed sign-ins, try again later")
	ErrInvalidCode        = errors.New("code is wrong")
	ErrCodeExpired        = errors.New("code has expired")
	ErrTooManyAttempts    = errors.New("too many wrong attempts, ask for a new code")
	ErrInvalidToken       = errors.New("token is invalid, expired or already used")
)

// ErrTokenReused reports that a refresh token spent earlier than the reuse grace came back, and that the whole
// family of sessions it belongs to has been revoked. Only SessionRepository.Rotate returns it, and no use case
// does: to a client the token is invalid, as any other refused token is.
var ErrTokenReused = errors.New("refresh token used again after it was spent; its family is revoked")
