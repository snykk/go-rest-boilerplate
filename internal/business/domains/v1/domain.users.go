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

type UserRepository interface {
	// Store inserts the user and returns the persisted row in a single
	// round-trip so callers don't need a follow-up GetByEmail (which
	// would orphan the INSERT if it failed).
	Store(ctx context.Context, inDom *UserDomain) (UserDomain, error)
	GetByEmail(ctx context.Context, inDom *UserDomain) (outDomain UserDomain, err error)
	ChangeActiveUser(ctx context.Context, inDom *UserDomain) (err error)
}
