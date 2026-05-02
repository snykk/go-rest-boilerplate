// Package users is the User bounded context. It owns user identity
// CRUD — create, look up, activate, soft-delete — and exposes nothing
// about authentication / sessions / tokens. The auth bounded context
// (internal/business/usecases/auth) calls into this package via the
// Usecase interface to look up or mutate user records.
package users

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// Usecase is the input boundary the auth context and HTTP handlers
// talk to when they need a user record. Each method's behavior is
// documented on its implementation file.
type Usecase interface {
	Store(ctx context.Context, in *domain.User) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	Activate(ctx context.Context, userID string) error
}
