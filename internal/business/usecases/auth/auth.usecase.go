// Package auth is the Auth bounded context. It owns credential
// verification, session lifecycle (access + refresh tokens), and OTP
// activation flow — everything that turns "this email + password"
// into "an authenticated request". The User bounded context
// (internal/business/usecases/users) is consulted via its Usecase
// interface for identity reads / writes; nothing in this package
// reaches into the User repository directly.
package auth

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

// Usecase is the input boundary the HTTP handler talks to. Each
// method's behavior is documented on its implementation file.
type Usecase interface {
	Register(ctx context.Context, in *entities.UserDomain) (entities.UserDomain, error)
	Login(ctx context.Context, email, password string) (LoginResult, error)
	SendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email, otpCode string) error
	Refresh(ctx context.Context, refreshToken string) (LoginResult, error)
	Logout(ctx context.Context, refreshToken string) error
}
