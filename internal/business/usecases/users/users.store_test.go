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

func TestStore_HashesPasswordAndNormalizesEmail(t *testing.T) {
	f := newFixture(t)
	in := &entities.UserDomain{
		Username: "newuser",
		Email:    "  NewUser@Example.COM ",
		Password: "Plaintext_123!",
		RoleID:   2,
	}

	// Capture the value passed to repo.Store so we can assert that
	// the password was hashed (not plaintext) and the email was
	// trimmed + lowercased before reaching the repo.
	f.repo.On("Store", mock.Anything, mock.MatchedBy(func(u *entities.UserDomain) bool {
		return u.Email == "newuser@example.com" &&
			u.Password != "Plaintext_123!" && u.Password != "" &&
			!u.CreatedAt.IsZero()
	})).Return(sampleUser(), nil).Once()

	out, err := f.usecase.Store(context.Background(), in)
	require.NoError(t, err)
	assert.NotEmpty(t, out.ID)
}

func TestStore_PropagatesRepoError(t *testing.T) {
	f := newFixture(t)
	repoErr := apperror.Conflict("username or email already exists")

	f.repo.On("Store", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).
		Return(entities.UserDomain{}, repoErr).Once()

	_, err := f.usecase.Store(context.Background(), &entities.UserDomain{
		Username: "dup", Email: "dup@example.com", Password: "Pwd_123!", RoleID: 2,
	})
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeConflict, domErr.Type,
		"domain error type from repo must pass through unchanged")
}
