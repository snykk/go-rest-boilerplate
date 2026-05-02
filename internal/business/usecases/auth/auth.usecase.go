// Package auth owns credential verification, session lifecycle
// (access + refresh tokens), OTP activation, and password lifecycle
// (change / forgot / reset).
package auth

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// Usecase is the input boundary the HTTP handler talks to.
type Usecase interface {
	// Register creates a new (inactive) account; activation requires the OTP flow.
	Register(ctx context.Context, in *domain.User) (domain.User, error)
	// Login validates credentials and returns a fresh access+refresh token pair.
	Login(ctx context.Context, email, password string) (LoginResult, error)
	// SendOTP issues a 6-digit code via email and stores it in Redis with TTL.
	SendOTP(ctx context.Context, email string) error
	// VerifyOTP consumes the code and activates the account; rate-limited per email.
	VerifyOTP(ctx context.Context, email, otpCode string) error
	// Refresh rotates a refresh token: mints a new pair and revokes the old jti.
	Refresh(ctx context.Context, refreshToken string) (LoginResult, error)
	// Logout deletes the refresh token's jti so it can't be used again.
	Logout(ctx context.Context, refreshToken string) error
	// ChangePassword swaps the password of the authenticated user.
	// Requires the current password to defeat session hijacking.
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
	// ForgotPassword starts a password-reset flow by emailing an opaque
	// token. Always returns nil for unknown emails to defeat user enumeration.
	ForgotPassword(ctx context.Context, email string) error
	// ResetPassword consumes a reset token and sets the new password.
	ResetPassword(ctx context.Context, token, newPassword string) error
}
