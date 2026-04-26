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
	// Store hashes the password, creates the user row, and returns the
	// persisted record. New accounts start with active=false until OTP
	// verification flips them on.
	Store(ctx context.Context, inDom *UserDomain) (outDom UserDomain, err error)
	// Login validates credentials and returns a fresh access+refresh
	// token pair. Wrong password and unknown email take the same wall
	// time to mask user enumeration.
	Login(ctx context.Context, inDom *UserDomain) (outDom UserDomain, err error)
	// SendOTP generates a 6-digit code, stores it in Redis with TTL,
	// and enqueues the email via the async mailer. The HTTP response
	// returns on enqueue, not on actual SMTP delivery.
	SendOTP(ctx context.Context, email string) error
	// VerifyOTP checks the supplied code against Redis, increments a
	// per-email attempt counter, and activates the account on success.
	// Lockout fires after OTP_MAX_ATTEMPTS failures.
	VerifyOTP(ctx context.Context, email string, userOTP string) error
	// GetByEmail returns the user, hitting the in-memory cache first
	// and coalescing concurrent misses through singleflight.
	GetByEmail(ctx context.Context, email string) (outDom UserDomain, err error)
	// Refresh verifies and rotates the refresh token, mints a new
	// access+refresh pair, and revokes the old jti.
	Refresh(ctx context.Context, refreshToken string) (outDom UserDomain, err error)
	// Logout revokes the supplied refresh token by deleting its jti
	// from Redis. Access tokens remain valid until their natural exp.
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
	// would orphan the INSERT if it failed). Duplicate username/email
	// surfaces as apperror.Conflict.
	Store(ctx context.Context, inDom *UserDomain) (UserDomain, error)
	// GetByEmail looks up a user by email, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByEmail(ctx context.Context, inDom *UserDomain) (outDomain UserDomain, err error)
	// GetByID looks up a user by primary key, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByID(ctx context.Context, id string) (UserDomain, error)
	// List returns users matching filter, paginated by offset/limit.
	// Limit is hard-capped server-side so a misbehaving caller can't
	// pull the whole table.
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]UserDomain, error)
	// ChangeActiveUser flips the active flag (used by the OTP-verify
	// flow) and stamps updated_at. No-op on soft-deleted rows.
	ChangeActiveUser(ctx context.Context, inDom *UserDomain) (err error)
	// SoftDelete sets deleted_at = NOW() so the row stays in the table
	// for audit/restore but stops matching default queries. Returns
	// apperror.NotFound if the row doesn't exist or is already deleted.
	SoftDelete(ctx context.Context, id string) error
}
