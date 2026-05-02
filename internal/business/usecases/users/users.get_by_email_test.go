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

func TestGetByEmail_CacheHit(t *testing.T) {
	f := newFixture(t)
	cached := sampleUser()
	f.rc.On("Get", "user/patrick@example.com").Return(cached).Once()

	out, err := f.usecase.GetByEmail(context.Background(), "patrick@example.com")
	require.NoError(t, err)
	assert.Equal(t, cached, out)
	// Repo never gets called on a cache hit.
}

func TestGetByEmail_CacheMissPopulatesCache(t *testing.T) {
	f := newFixture(t)
	expected := sampleUser()

	f.rc.On("Get", "user/patrick@example.com").Return(nil).Once()
	f.repo.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).
		Return(expected, nil).Once()
	f.rc.On("Set", "user/patrick@example.com", expected).Once()

	out, err := f.usecase.GetByEmail(context.Background(), "patrick@example.com")
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestGetByEmail_NotFound(t *testing.T) {
	f := newFixture(t)
	f.rc.On("Get", "user/ghost@example.com").Return(nil).Once()
	f.repo.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).
		Return(entities.UserDomain{}, apperror.NotFound("user not found")).Once()

	_, err := f.usecase.GetByEmail(context.Background(), "ghost@example.com")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeNotFound, domErr.Type)
}

func TestGetByEmail_NormalizesInputEmail(t *testing.T) {
	f := newFixture(t)
	// Mixed-case input must hash to the same lowercase cache key.
	f.rc.On("Get", "user/patrick@example.com").Return(sampleUser()).Once()

	_, err := f.usecase.GetByEmail(context.Background(), "  Patrick@Example.COM ")
	require.NoError(t, err)
}
