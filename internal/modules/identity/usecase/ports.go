// Package usecase runs the identity use cases and declares the ports they need from the outside world.
package usecase

import (
	"context"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

//go:generate go tool mockgen -source=ports.go -destination=ports_mock_test.go -package=usecase_test

// UserRepository stores and loads users.
type UserRepository interface {
	// Create stores a new user. It returns domain.ErrEmailTaken when another user has the same email.
	Create(ctx context.Context, user *domain.User) error
	// ByEmail returns the user with the given email, or domain.ErrNotFound.
	ByEmail(ctx context.Context, email domain.Email) (*domain.User, error)
	// ByID returns the user with the given ID, or domain.ErrNotFound.
	ByID(ctx context.Context, id string) (*domain.User, error)
	// Update loads the user with the given ID, applies change to it and saves every field, as one step that
	// concurrent updates of the same user wait for, and returns the saved user. It returns domain.ErrNotFound
	// for an unknown ID and change's error without saving anything.
	Update(ctx context.Context, id string, change func(user *domain.User) error) (*domain.User, error)
}

// SecretRepository stores the one secret of a kind, an e-mail verification code or a password reset token,
// that each user can hold at a time. A repository serves one kind; the secrets of two kinds never meet.
type SecretRepository interface {
	// Put stores secret as its user's only secret, replacing the one the user held, unless that one was created
	// less than cooldown before secret.CreatedAt: then it keeps the held secret, stores nothing and reports
	// stored false. Concurrent puts for one user apply one after the other, so of a burst within the cooldown
	// only the first is stored.
	Put(ctx context.Context, secret domain.Secret, cooldown time.Duration) (stored bool, err error)
	// Use loads the user's secret and runs check on it, as one step that concurrent uses of the same secret
	// wait for. What it stores depends on check's result: it deletes the secret when check returns nil, saves
	// the secret's Attempts when check returns domain.ErrInvalidCode, and keeps the secret as it is for any
	// other result. A secret refused as domain.ErrCodeExpired or domain.ErrTooManyAttempts therefore stays the
	// user's secret, refused by every later check, and Put counts the cooldown from it. It returns check's
	// error, and domain.ErrInvalidCode when the user holds no secret.
	Use(ctx context.Context, userID string, check func(secret *domain.Secret) error) error
	// ByHash returns the secret whose hash is the given one, or domain.ErrInvalidToken when there is none.
	ByHash(ctx context.Context, hash string) (domain.Secret, error)
}

// SessionRepository stores refresh-token sessions.
type SessionRepository interface {
	// Create stores a new session.
	Create(ctx context.Context, session domain.Session) error
	// Rotate spends the live session with the given token hash and stores next as its successor, as one step
	// that concurrent rotations of the same token wait for, and returns the user the session belongs to. The
	// caller leaves next's UserID and Family empty: they are the spent session's, so next joins its family. A
	// session is live when it is neither spent nor revoked and expires after now. Rotate, Revoke and RevokeAll
	// wait for one another when they are for the same user.
	//
	// It returns domain.ErrInvalidToken, changing nothing, for an unknown or revoked token, for an expired token
	// that was never spent, and for a token spent less than grace before now. For a token spent grace or more
	// before now, expired since or not, it revokes the whole family and returns the user the token belongs to
	// with domain.ErrTokenReused: the token was used again by someone who should not hold it. A token of a family
	// revoked that way is revoked, so presenting it again returns domain.ErrInvalidToken.
	Rotate(ctx context.Context, tokenHash string, next domain.Session, now time.Time,
		grace time.Duration) (userID string, err error)
	// Revoke revokes at now the family of the session with the given token hash: every session of the sign-in
	// the token belongs to, across its rotations, that is not revoked yet. It and Rotate serialise per user, so a
	// rotation of a token of the family either finishes first, and the session it stored is revoked too, or
	// starts after and finds its token revoked. It returns nil when no session has that hash.
	Revoke(ctx context.Context, tokenHash string, now time.Time) error
	// RevokeAll revokes every live session of the user at now. It and Rotate serialise per user: a rotation
	// running at the same time either finishes first, and the session it stored is revoked too, or starts
	// after and finds its token revoked. No session that was live before the call is live after it.
	RevokeAll(ctx context.Context, userID string, now time.Time) error
}

// UnitOfWork runs writes that succeed or fail together.
type UnitOfWork interface {
	// Do calls work with repositories that share one transaction. It commits when work returns nil, and rolls
	// back and returns work's error when it does not.
	Do(ctx context.Context, work func(repositories Repositories) error) error
}

// Repositories are the repositories of one unit of work.
type Repositories struct {
	Users         UserRepository
	Verifications SecretRepository
	Resets        SecretRepository
	Sessions      SessionRepository
}

// PasswordHasher hashes and compares passwords. The use cases hand it the text of a domain.Password, which it
// never logs or returns.
type PasswordHasher interface {
	// Hash returns the salted hash of password, or an error when it cannot make one.
	Hash(password string) (string, error)
	// Compare reports whether hash is the hash of password. A hash it cannot read does not match.
	Compare(hash, password string) bool
	// CompareDummy compares password against a hash of no user and discards the result, at the cost of a real
	// Compare, so the time a sign-in takes does not tell whether the account exists.
	CompareDummy(password string)
}

// Tokens issues access tokens.
type Tokens interface {
	// Issue returns an access token for the user with the given roles, and the time it expires. It returns an
	// error when it cannot sign one.
	Issue(userID string, roles []string) (token string, expiresAt time.Time, err error)
}

// Randomness provides the secrets the use cases hand out.
type Randomness interface {
	// Code returns a verification code: 6 digits from a cryptographic source.
	Code() string
	// Token returns an unguessable token: 32 random bytes from a cryptographic source, base64url encoded.
	Token() string
}

// Mail delivers the messages that carry a secret to a user.
type Mail interface {
	// Verification sends the verification code to the address. It never fails: a message that cannot be
	// delivered is the mailer's concern, and the user asks for a new code.
	Verification(ctx context.Context, email domain.Email, code string)
	// PasswordReset sends the password reset token to the address. It never fails: a message that cannot be
	// delivered is the mailer's concern, and the user asks for a new token.
	PasswordReset(ctx context.Context, email domain.Email, token string)
}

// Clock provides the current time.
type Clock interface {
	// Now returns the current time in UTC, at the precision the identity store keeps, so a stored time
	// reads back equal.
	Now() time.Time
}

// IDGenerator provides identifiers for new users, sessions and session families.
type IDGenerator interface {
	// NewID returns an identifier no other call returns, in the form the identity store accepts as an ID.
	NewID() string
}
