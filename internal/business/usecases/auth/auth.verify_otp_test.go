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

func TestVerifyOTP(t *testing.T) {
	tests := []struct {
		name  string
		email string
		code  string
		setup func(f *fixture)
		// wantErr / wantErrType paired because apperror.ErrTypeInternal
		// is the iota zero — a single sentinel would collide with that
		// type and silently pass.
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:  "happy path activates user and clears OTP + attempt keys",
			email: "patrick@example.com",
			code:  "123456",
			setup: func(f *fixture) {
				user := activeUser(t)
				user.Active = false
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
				f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "otp_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.redis.On("Get", mock.Anything, "user_otp:patrick@example.com").Return("123456", nil).Once()
				f.users.On("Activate", mock.Anything, user.ID).Return(nil).Once()
				f.redis.On("Del", mock.Anything, "user_otp:patrick@example.com").Return(nil).Once()
				f.redis.On("Del", mock.Anything, "otp_attempts:patrick@example.com").Return(nil).Once()
			},
		},
		{
			name:  "wrong code returns BadRequest after counting the attempt",
			email: "patrick@example.com",
			code:  "999999",
			setup: func(f *fixture) {
				user := activeUser(t)
				user.Active = false
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
				f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "otp_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.redis.On("Get", mock.Anything, "user_otp:patrick@example.com").Return("123456", nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
		{
			name:  "submitted code with wrong length is rejected (constant-time path)",
			email: "patrick@example.com",
			code:  "12",
			setup: func(f *fixture) {
				user := activeUser(t)
				user.Active = false
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
				f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "otp_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.redis.On("Get", mock.Anything, "user_otp:patrick@example.com").Return("123456", nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
		{
			name:  "lockout after exceeding OTPMaxAttempts surfaces as Forbidden",
			email: "patrick@example.com",
			code:  "123456",
			setup: func(f *fixture) {
				user := activeUser(t)
				user.Active = false
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
				// 6th attempt exceeds OTPMaxAttempts=5; Incr returns 6.
				// Forbidden is a distinct signal from BadRequest "wrong
				// code" so rate-limit / alerting can distinguish a
				// brute-force pattern from a typo.
				f.redis.On("Incr", mock.Anything, "otp_attempts:patrick@example.com").Return(int64(6), nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeForbidden,
		},
		{
			name:  "already-active account short-circuits with BadRequest",
			email: "patrick@example.com",
			code:  "123456",
			setup: func(f *fixture) {
				// Already active — early return, no Incr / Get / Activate.
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(activeUser(t), nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			err := f.usecase.VerifyOTP(context.Background(), tt.email, tt.code)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var domErr *apperror.DomainError
			require.True(t, errors.As(err, &domErr))
			assert.Equal(t, tt.wantErrType, domErr.Type)
		})
	}
}
