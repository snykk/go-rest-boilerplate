package requests

import (
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// General Request
type UserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=25"`
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72,strongpassword"`
}

// Mapping General Request to Domain User
func (user UserRequest) ToV1Domain() *domain.User {
	return &domain.User{
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
		RoleID:   2, // everyone who regis it's supposed to be users
	}
}

// Send OTP Request
type UserSendOTPRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

// Verify OTP Code
type UserVerifOTPRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
	Code  string `json:"code" validate:"required,numeric"`
}

// Login Request
type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=1,max=72"`
}

// Refresh Request — carries the refresh token minted at login.
type UserRefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Mapping Login Request to Domain User
func (u *UserLoginRequest) ToV1Domain() *domain.User {
	return &domain.User{
		Email:    u.Email,
		Password: u.Password,
	}
}
