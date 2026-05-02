package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSendOTP(t *testing.T) {
	tests := []struct {
		name  string
		email string
		setup func(f *fixture)
		// wantErr / wantErrType paired because apperror.ErrTypeInternal
		// is the iota zero — a single sentinel would collide with that
		// type and silently pass.
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:  "happy path mails OTP, sets cache, resets attempt counter",
			email: "patrick@example.com",
			setup: func(f *fixture) {
				user := activeUser(t)
				user.Active = false // SendOTP only valid for inactive accounts
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
				f.mailer.On("SendOTP", mock.AnythingOfType("string"), "patrick@example.com").Return(nil).Once()
				f.redis.On("Set", mock.Anything, "user_otp:patrick@example.com", mock.AnythingOfType("string")).Return(nil).Once()
				f.redis.On("Del", mock.Anything, "otp_attempts:patrick@example.com").Return(nil).Once()
			},
		},
		{
			name:  "already-active account short-circuits with BadRequest (no mailer / redis)",
			email: "patrick@example.com",
			setup: func(f *fixture) {
				// Active user — early return; no mailer / redis calls.
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(activeUser(t), nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
		{
			name:  "unknown email surfaces as NotFound from users.GetByEmail",
			email: "ghost@example.com",
			setup: func(f *fixture) {
				f.users.On("GetByEmail", mock.Anything, "ghost@example.com").
					Return(domain.User{}, apperror.NotFound("email not found")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			err := f.usecase.SendOTP(context.Background(), tt.email)

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
