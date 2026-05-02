package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVerifyOTP_HappyPath(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)
	user.Active = false

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
	f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(1), nil).Once()
	f.redis.On("Expire", mock.Anything, "otp_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
	f.redis.On("Get", mock.Anything, "user_otp:patrick@example.com").Return("123456", nil).Once()
	f.users.On("Activate", mock.Anything, user.ID).Return(nil).Once()
	f.redis.On("Del", mock.Anything, "user_otp:patrick@example.com").Return(nil).Once()
	f.redis.On("Del", mock.Anything, "otp_attempts:patrick@example.com").Return(nil).Once()

	require.NoError(t, f.usecase.VerifyOTP(context.Background(), "patrick@example.com", "123456"))
}

func TestVerifyOTP_WrongCode(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)
	user.Active = false

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
	f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(1), nil).Once()
	f.redis.On("Expire", mock.Anything, "otp_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
	f.redis.On("Get", mock.Anything, "user_otp:patrick@example.com").Return("123456", nil).Once()

	err := f.usecase.VerifyOTP(context.Background(), "patrick@example.com", "999999")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeBadRequest, domErr.Type)
}

func TestVerifyOTP_LockoutAfterMaxAttempts(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)
	user.Active = false

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
	// 6th attempt exceeds OTPMaxAttempts=5; Incr returns 6.
	f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(6), nil).Once()

	err := f.usecase.VerifyOTP(context.Background(), "patrick@example.com", "123456")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeForbidden, domErr.Type,
		"lockout must surface as Forbidden (distinct signal from BadRequest 'wrong code') "+
			"so rate-limit + alerting can distinguish brute-force pattern")
}

func TestVerifyOTP_AlreadyActiveAccount(t *testing.T) {
	f := newFixture(t)
	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(activeUser(t), nil).Once()
	// Already active — early return, no Incr / Get / Activate.

	err := f.usecase.VerifyOTP(context.Background(), "patrick@example.com", "123456")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeBadRequest, domErr.Type)
}
