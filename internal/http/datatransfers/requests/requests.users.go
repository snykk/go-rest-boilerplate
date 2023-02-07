package requests

import (
	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
)

// General Request
type UserRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,containsany=!@#$%^&*()?"`
}

// Mapping General Request to Domain User
func (user UserRequest) ToDomain() *domains.UserDomain {
	return &domains.UserDomain{
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
		RoleID:   2, // everyone who regis it's supposed to be users
	}
}

// Send OTP Request
type UserSendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// Verify OTP Code
type UserVerifOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,numeric"`
}

// Login Request
type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,containsany=!@#$%^&*()?"`
}

// Mapping Login Request to Domain User
func (u *UserLoginRequest) ToDomain() *domains.UserDomain {
	return &domains.UserDomain{
		Email:    u.Email,
		Password: u.Password,
	}
}
