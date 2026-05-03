// Package users owns user identity CRUD: create, look up, activate,
// soft-delete, and password rotation.
package users

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// Usecase is the input boundary. Each method takes a Request struct
// and (when it has data to return) yields a Response struct, so
// adding fields stays backward-compatible across versions.
type Usecase interface {
	// Store builds a fresh User (normalized email, hashed password) and
	// persists it; returns the inserted row including DB-generated ID.
	Store(ctx context.Context, req StoreRequest) (StoreResponse, error)
	// GetByEmail returns the user with the given email; cache-first
	// lookup with singleflight coalescing on miss.
	GetByEmail(ctx context.Context, req GetByEmailRequest) (GetByEmailResponse, error)
	// GetByID returns the user with the given primary key; bypasses cache.
	GetByID(ctx context.Context, req GetByIDRequest) (GetByIDResponse, error)
	// Activate flips the user's active flag (called by the OTP-verify flow).
	Activate(ctx context.Context, req ActivateRequest) error
	// UpdatePassword swaps the user's password (already hashed by the
	// caller via domain.User.ChangePassword) and stamps password_changed_at.
	UpdatePassword(ctx context.Context, req UpdatePasswordRequest) error
}

// Request / Response types for the Usecase boundary. Adding a field
// to a struct doesn't break callers, while adding a parameter to a
// method signature does — Uncle Bob's "Input/Output Boundary"
// recommendation, made concrete.
type (
	StoreRequest struct {
		User *domain.User
	}
	StoreResponse struct {
		User domain.User
	}

	GetByEmailRequest struct {
		Email string
	}
	GetByEmailResponse struct {
		User domain.User
	}

	GetByIDRequest struct {
		ID string
	}
	GetByIDResponse struct {
		User domain.User
	}

	ActivateRequest struct {
		UserID string
	}

	UpdatePasswordRequest struct {
		User *domain.User
	}
)
