// Package _interface holds the gateway abstractions for every domain
// in the repositories layer. The package name is "_interface" because
// "interface" is a Go reserved keyword and can't be used as an
// identifier directly; the leading underscore keeps it a valid
// identifier without changing the conceptual meaning.
//
// Concrete adapters (postgres/, future mongo/, etc.) implement these
// interfaces and live as siblings of this package. The use case
// layer depends only on this package — never on a concrete adapter
// — so swapping the storage engine doesn't ripple into business
// code.
package _interface

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// UserListFilter narrows down UserRepository.List() results. Each
// field is optional; the empty value means "no filter on this
// dimension". Domain-prefixed (UserListFilter, future
// ProductListFilter) so multiple filter types can co-exist in this
// shared package without collision.
type UserListFilter struct {
	RoleID         int  // 0 = any role
	ActiveOnly     bool // true = only active=true users
	IncludeDeleted bool // false (default) = WHERE deleted_at IS NULL
}

// UserRepository is the gateway for loading and persisting users.
type UserRepository interface {
	// Store inserts the user and returns the persisted row in a single
	// round-trip so callers don't need a follow-up GetByEmail (which
	// would orphan the INSERT if it failed). Duplicate username/email
	// surfaces as apperror.Conflict.
	Store(ctx context.Context, in *domain.User) (domain.User, error)
	// GetByEmail looks up a user by email, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByEmail(ctx context.Context, in *domain.User) (out domain.User, err error)
	// GetByID looks up a user by primary key, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByID(ctx context.Context, id string) (domain.User, error)
	// List returns users matching filter, paginated by offset/limit.
	// Limit is hard-capped server-side so a misbehaving caller can't
	// pull the whole table.
	List(ctx context.Context, filter UserListFilter, offset, limit int) ([]domain.User, error)
	// ChangeActiveUser flips the active flag (used by the OTP-verify
	// flow) and stamps updated_at. No-op on soft-deleted rows.
	ChangeActiveUser(ctx context.Context, in *domain.User) (err error)
	// UpdatePassword swaps the bcrypt hash and stamps password_changed_at +
	// updated_at. Returns apperror.NotFound if the user is missing/soft-deleted.
	UpdatePassword(ctx context.Context, in *domain.User) error
	// SoftDelete sets deleted_at = NOW() so the row stays in the table
	// for audit/restore but stops matching default queries. Returns
	// apperror.NotFound if the row doesn't exist or is already deleted.
	SoftDelete(ctx context.Context, id string) error
}
