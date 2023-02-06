package requests

import (
	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
)

type UserRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,containsany=!@#$%^&*()?"`
}

func (user UserRequest) ToDomain() *domains.UserDomain {
	return &domains.UserDomain{
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
	}
}
