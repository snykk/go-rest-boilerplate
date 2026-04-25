package v1

import (
	"context"
	"time"
)

type UserDomain struct {
	ID           string
	Username     string
	Email        string
	Password     string
	Active       bool
	Token        string // access token
	RefreshToken string
	RoleID       int
	CreatedAt    time.Time
	UpdatedAt    *time.Time
	DeletedAt    *time.Time
}

type UserUsecase interface {
	Store(ctx context.Context, inDom *UserDomain) (outDom UserDomain, err error)
	Login(ctx context.Context, inDom *UserDomain) (outDom UserDomain, err error)
	SendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email string, userOTP string) error
	GetByEmail(ctx context.Context, email string) (outDom UserDomain, err error)
	Refresh(ctx context.Context, refreshToken string) (outDom UserDomain, err error)
	Logout(ctx context.Context, refreshToken string) error
}

// ListFilter narrows down List() results. Each field is optional;
// the empty value means "no filter on this dimension".
type ListFilter struct {
	RoleID         int  // 0 = any role
	ActiveOnly     bool // true = only active=true users
	IncludeDeleted bool // false (default) = WHERE deleted_at IS NULL
}

type UserRepository interface {
	// Store inserts the user and returns the persisted row in a single
	// round-trip so callers don't need a follow-up GetByEmail (which
	// would orphan the INSERT if it failed).
	Store(ctx context.Context, inDom *UserDomain) (UserDomain, error)
	GetByEmail(ctx context.Context, inDom *UserDomain) (outDomain UserDomain, err error)
	GetByID(ctx context.Context, id string) (UserDomain, error)
	// List returns users matching filter, paginated by offset/limit.
	// Caller is responsible for clamping limit to a sane maximum.
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]UserDomain, error)
	ChangeActiveUser(ctx context.Context, inDom *UserDomain) (err error)
	// SoftDelete sets deleted_at = NOW() so the row stays in the table
	// for audit/restore but stops matching default queries.
	SoftDelete(ctx context.Context, id string) error
}
