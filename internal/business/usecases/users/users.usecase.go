// Package users owns user identity CRUD: create, look up, activate,
// soft-delete, and password rotation.
package users

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// Usecase is the input boundary the auth context and HTTP handlers
// talk to when they need a user record.
type Usecase interface {
	// Store builds a fresh User (normalized email, hashed password) and
	// persists it; returns the inserted row including DB-generated ID.
	Store(ctx context.Context, in *domain.User) (domain.User, error)
	// GetByEmail returns the user with the given email; cache-first
	// lookup with singleflight coalescing on miss.
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	// GetByID returns the user with the given primary key; bypasses cache.
	GetByID(ctx context.Context, id string) (domain.User, error)
	// Activate flips the user's active flag (called by the OTP-verify flow).
	Activate(ctx context.Context, userID string) error
	// UpdatePassword swaps the user's password (already hashed by the
	// caller via domain.User.ChangePassword) and stamps password_changed_at.
	UpdatePassword(ctx context.Context, user *domain.User) error
}
