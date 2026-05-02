package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByID_PassesThroughToRepo(t *testing.T) {
	f := newFixture(t)
	expected := sampleUser()
	f.repo.On("GetByID", context.Background(), expected.ID).Return(expected, nil).Once()

	out, err := f.usecase.GetByID(context.Background(), expected.ID)
	require.NoError(t, err)
	assert.Equal(t, expected, out, "repo result must pass through unchanged")
}

func TestGetByID_PreservesNotFound(t *testing.T) {
	f := newFixture(t)
	f.repo.On("GetByID", context.Background(), "missing").
		Return(entities.UserDomain{}, apperror.NotFound("user not found")).Once()

	_, err := f.usecase.GetByID(context.Background(), "missing")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeNotFound, domErr.Type)
}

func TestGetByID_WrapsRawRepoError(t *testing.T) {
	f := newFixture(t)
	f.repo.On("GetByID", context.Background(), "any").
		Return(entities.UserDomain{}, errors.New("connection refused")).Once()

	_, err := f.usecase.GetByID(context.Background(), "any")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr),
		"raw errors must be wrapped into DomainError so HTTP layer can map them")
	assert.Equal(t, apperror.ErrTypeInternal, domErr.Type)
}
