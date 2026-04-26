// Package repositories owns the gateway abstractions for the data
// layer. Subpackages (postgres/, mongo/, etc.) provide concrete
// implementations of these interfaces.
//
// The use case layer depends on this package — never on a concrete
// adapter — so swapping the storage engine doesn't ripple into
// business code.
package repositories

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

// ListFilter narrows down List() results. Each field is optional;
// the empty value means "no filter on this dimension".
type ListFilter struct {
	RoleID         int  // 0 = any role
	ActiveOnly     bool // true = only active=true users
	IncludeDeleted bool // false (default) = WHERE deleted_at IS NULL
}

// UserRepository is the gateway for loading and persisting users.
// Implementations live alongside the storage engine they target
// (e.g., repositories/postgres/v1).
type UserRepository interface {
	// Store inserts the user and returns the persisted row in a single
	// round-trip so callers don't need a follow-up GetByEmail (which
	// would orphan the INSERT if it failed). Duplicate username/email
	// surfaces as apperror.Conflict.
	Store(ctx context.Context, inDom *entities.UserDomain) (entities.UserDomain, error)
	// GetByEmail looks up a user by email, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByEmail(ctx context.Context, inDom *entities.UserDomain) (outDomain entities.UserDomain, err error)
	// GetByID looks up a user by primary key, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByID(ctx context.Context, id string) (entities.UserDomain, error)
	// List returns users matching filter, paginated by offset/limit.
	// Limit is hard-capped server-side so a misbehaving caller can't
	// pull the whole table.
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]entities.UserDomain, error)
	// ChangeActiveUser flips the active flag (used by the OTP-verify
	// flow) and stamps updated_at. No-op on soft-deleted rows.
	ChangeActiveUser(ctx context.Context, inDom *entities.UserDomain) (err error)
	// SoftDelete sets deleted_at = NOW() so the row stays in the table
	// for audit/restore but stops matching default queries. Returns
	// apperror.NotFound if the row doesn't exist or is already deleted.
	SoftDelete(ctx context.Context, id string) error
}
