package auth_test

import (
	"context"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegister_DelegatesToUsersStore(t *testing.T) {
	f := newFixture(t)
	stored := entities.UserDomain{ID: "u-1", Email: "x@y.com"}

	f.users.On("Store", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).
		Return(stored, nil).Once()

	out, err := f.usecase.Register(context.Background(), &entities.UserDomain{
		Username: "x", Email: "x@y.com", Password: "Pwd_123!", RoleID: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, stored, out, "Register must return whatever users.Store returned, unchanged")
}
