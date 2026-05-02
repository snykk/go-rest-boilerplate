package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestActivate_FlipsActiveFlag(t *testing.T) {
	f := newFixture(t)
	// Verify the repo is called with Active=true and the right ID.
	f.repo.On("ChangeActiveUser", mock.Anything, mock.MatchedBy(func(u *entities.UserDomain) bool {
		return u.ID == "user-123" && u.Active == true
	})).Return(nil).Once()

	require.NoError(t, f.usecase.Activate(context.Background(), "user-123"))
}

func TestActivate_WrapsRepoError(t *testing.T) {
	f := newFixture(t)
	f.repo.On("ChangeActiveUser", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).
		Return(errors.New("deadlock")).Once()

	err := f.usecase.Activate(context.Background(), "user-123")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeInternal, domErr.Type)
}
