// Package domain holds the enterprise entities — the innermost circle
// of Clean Architecture. To keep this layer stable as outer layers
// (HTTP, DB, framework) evolve, domain depends only on:
//
//   - the standard library
//   - golang.org/x/crypto/bcrypt (a stable cryptographic primitive,
//     treated like an extension of the standard library)
//
// Domain MUST NOT import internal/ or pkg/ packages — that would
// invert the dependency rule (inner depending on outer).
//
// Timestamps are stamped in UTC. Display timezones (e.g. WIB / GMT+7)
// are a presentation concern handled by outer layers, not the domain.
package domain

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role identifiers. Authorization decisions like IsAdmin() are domain
// logic, so the role IDs live with the domain — not in a transport-
// or persistence-adjacent constants package.
const (
	RoleAdmin = 1
	RoleUser  = 2
)

// Domain errors as plain sentinels so callers can compare via
// errors.Is without coupling to any error envelope. The transport
// layer wraps these into HTTP-shaped responses; persistence wraps
// them into DB-shaped responses.
var (
	ErrEmptyUsername = errors.New("username cannot be empty")
	ErrEmptyEmail    = errors.New("email cannot be empty")
	ErrEmptyPassword = errors.New("password cannot be empty")
)

// User is the domain entity for a registered account. Password
// always carries the bcrypt hash post-construction — plaintext only
// exists transiently inside NewUser.
type User struct {
	ID        string
	Username  string
	Email     string
	Password  string
	Active    bool
	RoleID    int
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

// NewUser builds a fresh User from registration input. Email is
// normalized, password is hashed at the supplied bcrypt cost, and
// CreatedAt is stamped in the canonical timezone.
//
// bcryptCost is a parameter (not read from config) so the domain
// stays free of configuration concerns; the caller injects it. Out-
// of-range values fall back to bcrypt.DefaultCost so a misconfigured
// outer layer can't make this panic.
func NewUser(username, email, plainPassword string, roleID, bcryptCost int) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrEmptyUsername
	}
	if plainPassword == "" {
		return nil, ErrEmptyPassword
	}
	email = NormalizeEmail(email)
	if email == "" {
		return nil, ErrEmptyEmail
	}

	if bcryptCost < bcrypt.MinCost || bcryptCost > bcrypt.MaxCost {
		bcryptCost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcryptCost)
	if err != nil {
		return nil, err
	}

	return &User{
		Username:  username,
		Email:     email,
		Password:  string(hash),
		RoleID:    roleID,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// NormalizeEmail trims whitespace and lowercases the address so
// "User@Example.com " and "user@example.com" hash to the same lookup
// key, query the same DB row, and trip the same uniqueness violation.
// RFC 5321 says the local part is technically case-sensitive, but
// every consumer-grade mail provider treats it case-insensitively.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// Activate flips the user to active and stamps UpdatedAt. Pointer
// receiver because this mutates state.
func (u *User) Activate() {
	u.Active = true
	now := time.Now().UTC()
	u.UpdatedAt = &now
}

// VerifyPassword returns true iff plain hashes to u.Password under
// bcrypt. Value receiver — pure read.
func (u User) VerifyPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
}

// IsAdmin reports whether the user's role grants admin privileges.
// A method (not a bare comparison at call sites) so the rule lives in
// one place — change RoleAdmin once and every caller follows.
func (u User) IsAdmin() bool { return u.RoleID == RoleAdmin }
