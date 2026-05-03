package requests

import (
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// RegisterRequest is the body for POST /auth/register.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=25"`
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72,strongpassword"`
}

func (r RegisterRequest) ToV1Domain() *domain.User {
	return &domain.User{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
		RoleID:   2, // everyone who registers is a regular user
	}
}

// SendOTPRequest is the body for POST /auth/send-otp.
type SendOTPRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

// VerifyOTPRequest is the body for POST /auth/verify-otp.
type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
	Code  string `json:"code" validate:"required,numeric"`
}

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=1,max=72"`
}

func (r *LoginRequest) ToV1Domain() *domain.User {
	return &domain.User{
		Email:    r.Email,
		Password: r.Password,
	}
}

// RefreshRequest is the body for POST /auth/refresh and POST /auth/logout —
// both consume a refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest is the body for PUT /auth/password/change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=1,max=72"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=72,strongpassword"`
}

// ForgotPasswordRequest is the body for POST /auth/password/forgot.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

// ResetPasswordRequest is the body for POST /auth/password/reset.
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72,strongpassword"`
}
