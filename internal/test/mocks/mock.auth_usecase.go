// Hand-written mock for auth.Usecase, mirroring the testify/mock
// style used by the rest of the project. Compile-time signature
// matching is enforced because handler tests instantiate this where
// auth.Usecase is expected — drift triggers a build failure.

package mocks

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	mock "github.com/stretchr/testify/mock"
)

type AuthUsecase struct {
	mock.Mock
}

func (_m *AuthUsecase) Register(ctx context.Context, req auth.RegisterRequest) (auth.RegisterResponse, error) {
	ret := _m.Called(ctx, req)
	return ret.Get(0).(auth.RegisterResponse), ret.Error(1)
}

func (_m *AuthUsecase) Login(ctx context.Context, req auth.LoginRequest) (auth.LoginResponse, error) {
	ret := _m.Called(ctx, req)
	return ret.Get(0).(auth.LoginResponse), ret.Error(1)
}

func (_m *AuthUsecase) SendOTP(ctx context.Context, req auth.SendOTPRequest) error {
	return _m.Called(ctx, req).Error(0)
}

func (_m *AuthUsecase) VerifyOTP(ctx context.Context, req auth.VerifyOTPRequest) error {
	return _m.Called(ctx, req).Error(0)
}

func (_m *AuthUsecase) Refresh(ctx context.Context, req auth.RefreshRequest) (auth.LoginResponse, error) {
	ret := _m.Called(ctx, req)
	return ret.Get(0).(auth.LoginResponse), ret.Error(1)
}

func (_m *AuthUsecase) Logout(ctx context.Context, req auth.LogoutRequest) error {
	return _m.Called(ctx, req).Error(0)
}

func (_m *AuthUsecase) ChangePassword(ctx context.Context, req auth.ChangePasswordRequest) error {
	return _m.Called(ctx, req).Error(0)
}

func (_m *AuthUsecase) ForgotPassword(ctx context.Context, req auth.ForgotPasswordRequest) error {
	return _m.Called(ctx, req).Error(0)
}

func (_m *AuthUsecase) ResetPassword(ctx context.Context, req auth.ResetPasswordRequest) error {
	return _m.Called(ctx, req).Error(0)
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
