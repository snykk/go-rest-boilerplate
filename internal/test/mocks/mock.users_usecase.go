// Hand-written mock for users.Usecase, using testify/mock to match
// the rest of the project's mock style (mockery would generate the
// same shape). Lives here rather than under an auto-gen block so
// changes to the Usecase interface produce a compile-time mismatch
// against this file — the test author has to update this mock too,
// no silent drift.

package mocks

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	mock "github.com/stretchr/testify/mock"
)

// UsersUsecase is the mock for users.Usecase.
type UsersUsecase struct {
	mock.Mock
}

// Store provides a mock function with given fields: ctx, in
func (_m *UsersUsecase) Store(ctx context.Context, in *domain.User) (domain.User, error) {
	ret := _m.Called(ctx, in)

	var r0 domain.User
	if rf, ok := ret.Get(0).(func(context.Context, *domain.User) domain.User); ok {
		r0 = rf(ctx, in)
	} else {
		r0 = ret.Get(0).(domain.User)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *domain.User) error); ok {
		r1 = rf(ctx, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByEmail provides a mock function with given fields: ctx, email
func (_m *UsersUsecase) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	ret := _m.Called(ctx, email)

	var r0 domain.User
	if rf, ok := ret.Get(0).(func(context.Context, string) domain.User); ok {
		r0 = rf(ctx, email)
	} else {
		r0 = ret.Get(0).(domain.User)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID provides a mock function with given fields: ctx, id
func (_m *UsersUsecase) GetByID(ctx context.Context, id string) (domain.User, error) {
	ret := _m.Called(ctx, id)

	var r0 domain.User
	if rf, ok := ret.Get(0).(func(context.Context, string) domain.User); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(domain.User)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Activate provides a mock function with given fields: ctx, userID
func (_m *UsersUsecase) Activate(ctx context.Context, userID string) error {
	ret := _m.Called(ctx, userID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewUsersUsecase interface {
	mock.TestingT
	Cleanup(func())
}

// NewUsersUsecase creates a new instance of UsersUsecase. It also
// registers a testing interface on the mock and a cleanup function
// to assert the mock's expectations.
func NewUsersUsecase(t mockConstructorTestingTNewUsersUsecase) *UsersUsecase {
	m := &UsersUsecase{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}
