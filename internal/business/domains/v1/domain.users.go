package v1

import (
	"context"
	"time"
)

type UserDomain struct {
	ID        string
	Username  string
	Email     string
	Password  string
	Active    bool
	Token     string
	RoleID    int
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

type UserUsecase interface {
	Store(ctx context.Context, inDom *UserDomain) (outDom UserDomain, statusCode int, err error)
	Login(ctx context.Context, inDom *UserDomain) (outDom UserDomain, statusCode int, err error)
	SendOTP(ctx context.Context, email string) (otpCode string, statusCode int, err error)
	VerifOTP(ctx context.Context, email string, userOTP string, otpRedis string) (statusCode int, err error)
	ActivateUser(ctx context.Context, email string) (statusCode int, err error)
	GetByEmail(ctx context.Context, email string) (outDom UserDomain, statusCode int, err error)
}

type UserRepository interface {
	Store(ctx context.Context, inDom *UserDomain) (err error)
	GetByEmail(ctx context.Context, inDom *UserDomain) (outDomain UserDomain, err error)
	ChangeActiveUser(ctx context.Context, inDom *UserDomain) (err error)
}
