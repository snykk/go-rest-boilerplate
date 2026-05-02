// Hand-written mock for auth.Usecase, mirroring the testify/mock
// style used by the rest of the project. Compile-time signature
// matching is enforced because the package importing this mock
// (handler tests) accepts auth.Usecase — so any drift triggers a
// build failure rather than silent runtime drift.

package mocks

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	mock "github.com/stretchr/testify/mock"
)

type AuthUsecase struct {
	mock.Mock
}

func (_m *AuthUsecase) Register(ctx context.Context, in *domain.User) (domain.User, error) {
	ret := _m.Called(ctx, in)
	return ret.Get(0).(domain.User), ret.Error(1)
}

func (_m *AuthUsecase) Login(ctx context.Context, email, password string) (auth.LoginResult, error) {
	ret := _m.Called(ctx, email, password)
	return ret.Get(0).(auth.LoginResult), ret.Error(1)
}

func (_m *AuthUsecase) SendOTP(ctx context.Context, email string) error {
	return _m.Called(ctx, email).Error(0)
}

func (_m *AuthUsecase) VerifyOTP(ctx context.Context, email, otpCode string) error {
	return _m.Called(ctx, email, otpCode).Error(0)
}

func (_m *AuthUsecase) Refresh(ctx context.Context, refreshToken string) (auth.LoginResult, error) {
	ret := _m.Called(ctx, refreshToken)
	return ret.Get(0).(auth.LoginResult), ret.Error(1)
}

func (_m *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	return _m.Called(ctx, refreshToken).Error(0)
}

func (_m *AuthUsecase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return _m.Called(ctx, userID, currentPassword, newPassword).Error(0)
}

func (_m *AuthUsecase) ForgotPassword(ctx context.Context, email string) error {
	return _m.Called(ctx, email).Error(0)
}

func (_m *AuthUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	return _m.Called(ctx, token, newPassword).Error(0)
}

type mockConstructorTestingTNewAuthUsecase interface {
	mock.TestingT
	Cleanup(func())
}

func NewAuthUsecase(t mockConstructorTestingTNewAuthUsecase) *AuthUsecase {
	m := &AuthUsecase{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}
