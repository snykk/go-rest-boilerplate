package auth_test

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

func TestSendOTP_HappyPath(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)
	user.Active = false // SendOTP is only valid for inactive accounts

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
	f.mailer.On("SendOTP", mock.AnythingOfType("string"), "patrick@example.com").Return(nil).Once()
	f.redis.On("Set", mock.Anything, "user_otp:patrick@example.com", mock.AnythingOfType("string")).Return(nil).Once()
	f.redis.On("Del", mock.Anything, "otp_attempts:patrick@example.com").Return(nil).Once()

	require.NoError(t, f.usecase.SendOTP(context.Background(), "patrick@example.com"))
}

func TestSendOTP_AlreadyActive(t *testing.T) {
	f := newFixture(t)
	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(activeUser(t), nil).Once()
	// Active user — early return, no mailer / redis calls.

	err := f.usecase.SendOTP(context.Background(), "patrick@example.com")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeBadRequest, domErr.Type)
}

func TestSendOTP_EmailNotRegistered(t *testing.T) {
	f := newFixture(t)
	f.users.On("GetByEmail", mock.Anything, "ghost@example.com").
		Return(entities.UserDomain{}, apperror.NotFound("email not found")).Once()

	err := f.usecase.SendOTP(context.Background(), "ghost@example.com")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeNotFound, domErr.Type)
}
