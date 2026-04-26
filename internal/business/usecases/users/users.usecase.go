// Package users is the User bounded context. It owns user identity
// CRUD — create, look up, activate, soft-delete — and exposes nothing
// about authentication / sessions / tokens. The auth bounded context
// (internal/business/usecases/auth) calls into this package via the
// Usecase interface to look up or mutate user records.
package users

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

// Usecase is the input boundary the auth context and HTTP handlers
// talk to when they need a user record. Each method's behavior is
// documented on its implementation file.
type Usecase interface {
	Store(ctx context.Context, in *entities.UserDomain) (entities.UserDomain, error)
	GetByEmail(ctx context.Context, email string) (entities.UserDomain, error)
	GetByID(ctx context.Context, id string) (entities.UserDomain, error)
	Activate(ctx context.Context, userID string) error
}
