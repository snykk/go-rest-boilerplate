// Package auth owns credential verification, session lifecycle
// (access + refresh tokens), OTP activation, and password lifecycle
// (change / forgot / reset).
package auth

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// Usecase is the input boundary the HTTP handler talks to. Each
// method takes a Request struct and (when it has data to return)
// yields a Response struct, so adding fields stays backward-
// compatible across versions.
type Usecase interface {
	// Register creates a new (inactive) account; activation requires the OTP flow.
	Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error)
	// Login validates credentials and returns a fresh access+refresh token pair.
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
	// SendOTP issues a 6-digit code via email and stores it in Redis with TTL.
	SendOTP(ctx context.Context, req SendOTPRequest) error
	// VerifyOTP consumes the code and activates the account; rate-limited per email.
	VerifyOTP(ctx context.Context, req VerifyOTPRequest) error
	// Refresh rotates a refresh token: mints a new pair and revokes the old jti.
	Refresh(ctx context.Context, req RefreshRequest) (LoginResponse, error)
	// Logout deletes the refresh token's jti so it can't be used again.
	Logout(ctx context.Context, req LogoutRequest) error
	// ChangePassword swaps the password of the authenticated user.
	// Requires the current password to defeat session hijacking.
	ChangePassword(ctx context.Context, req ChangePasswordRequest) error
	// ForgotPassword starts a password-reset flow by emailing an opaque
	// token. Always returns nil for unknown emails to defeat user enumeration.
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	// ResetPassword consumes a reset token and sets the new password.
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
}

// Request / Response types for the Usecase boundary. Adding a field
// to a struct doesn't break callers, while adding a parameter to a
// method signature does — Uncle Bob's "Input/Output Boundary"
// recommendation, made concrete.
type (
	RegisterRequest struct {
		User *domain.User
	}
	RegisterResponse struct {
		User domain.User
	}

	LoginRequest struct {
		Email    string
		Password string
	}

	LoginResponse struct {
		User         domain.User
		AccessToken  string
		RefreshToken string
	}

	SendOTPRequest struct {
		Email string
	}

	VerifyOTPRequest struct {
		Email   string
		OTPCode string
	}

	RefreshRequest struct {
		RefreshToken string
	}

	LogoutRequest struct {
		RefreshToken string
	}

	ChangePasswordRequest struct {
		UserID          string
		CurrentPassword string
		NewPassword     string
	}

	ForgotPasswordRequest struct {
		Email string
	}

	ResetPasswordRequest struct {
		Token       string
		NewPassword string
	}
)
