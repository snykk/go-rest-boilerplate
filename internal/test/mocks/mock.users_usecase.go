// Hand-written mock for users.Usecase, using testify/mock to match
// the rest of the project's mock style. Compile-time signature match
// is enforced because tests instantiate this where users.Usecase is
// expected — drift triggers a build failure rather than runtime
// surprise.

package mocks

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	mock "github.com/stretchr/testify/mock"
)

// UsersUsecase is the mock for users.Usecase.
type UsersUsecase struct {
	mock.Mock
}

func (_m *UsersUsecase) Store(ctx context.Context, req users.StoreRequest) (users.StoreResponse, error) {
	ret := _m.Called(ctx, req)
	return ret.Get(0).(users.StoreResponse), ret.Error(1)
}

func (_m *UsersUsecase) GetByEmail(ctx context.Context, req users.GetByEmailRequest) (users.GetByEmailResponse, error) {
	ret := _m.Called(ctx, req)
	return ret.Get(0).(users.GetByEmailResponse), ret.Error(1)
}

func (_m *UsersUsecase) GetByID(ctx context.Context, req users.GetByIDRequest) (users.GetByIDResponse, error) {
	ret := _m.Called(ctx, req)
	return ret.Get(0).(users.GetByIDResponse), ret.Error(1)
}

func (_m *UsersUsecase) UpdatePassword(ctx context.Context, req users.UpdatePasswordRequest) error {
	return _m.Called(ctx, req).Error(0)
}

func (_m *UsersUsecase) Activate(ctx context.Context, req users.ActivateRequest) error {
	return _m.Called(ctx, req).Error(0)
}

type mockConstructorTestingTNewUsersUsecase interface {
	mock.TestingT
	Cleanup(func())
}

func NewUsersUsecase(t mockConstructorTestingTNewUsersUsecase) *UsersUsecase {
	m := &UsersUsecase{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}
