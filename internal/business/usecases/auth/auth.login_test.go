package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func samplePair() jwt.TokenPair {
	return jwt.TokenPair{
		AccessToken:      "access-tok",
		RefreshToken:     "refresh-tok",
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		AccessJTI:        "access-jti",
		RefreshJTI:       "refresh-jti",
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		setup    func(f *fixture)
		// wantErr / wantErrType paired because apperror.ErrTypeInternal
		// is the iota zero — a single sentinel would collide with that
		// type and silently pass.
		wantErr     bool
		wantErrType apperror.ErrorType
		// wantErrMsg is asserted only for the unauthorized cases — the
		// timing-attack mitigation requires identical error text on
		// "wrong password" and "unknown email", so we pin the message
		// to catch regressions that leak which path was taken.
		wantErrMsg string
	}{
		{
			name:     "happy path issues token pair, persists refresh JTI, clears attempt counter",
			email:    "patrick@example.com",
			password: "Pwd_123!",
			setup: func(f *fixture) {
				user := activeUser(t)
				f.redis.On("Incr", mock.Anything, "login_attempts:patrick@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "login_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.users.On("GetByEmail", mock.Anything, users.GetByEmailRequest{Email: "patrick@example.com"}).Return(users.GetByEmailResponse{User: user}, nil).Once()
				f.jwt.On("GenerateTokenPair", user.ID, false, user.Email).Return(samplePair(), nil).Once()
				f.redis.On("Set", mock.Anything, "refresh:refresh-jti", "refresh-jti").Return(nil).Once()
				f.redis.On("Expire", mock.Anything, "refresh:refresh-jti", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.redis.On("Del", mock.Anything, "login_attempts:patrick@example.com").Return(nil).Once()
			},
		},
		{
			name:     "wrong password returns Unauthorized; counter is not cleared",
			email:    "patrick@example.com",
			password: "wrong-password",
			setup: func(f *fixture) {
				f.redis.On("Incr", mock.Anything, "login_attempts:patrick@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "login_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.users.On("GetByEmail", mock.Anything, users.GetByEmailRequest{Email: "patrick@example.com"}).Return(users.GetByEmailResponse{User: activeUser(t)}, nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
			wantErrMsg:  "invalid email or password",
		},
		{
			name:     "inactive account returns Forbidden",
			email:    "patrick@example.com",
			password: "Pwd_123!",
			setup: func(f *fixture) {
				user := activeUser(t)
				user.Active = false
				f.redis.On("Incr", mock.Anything, "login_attempts:patrick@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "login_attempts:patrick@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.users.On("GetByEmail", mock.Anything, users.GetByEmailRequest{Email: "patrick@example.com"}).Return(users.GetByEmailResponse{User: user}, nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeForbidden,
		},
		{
			name:     "unknown email surfaces as invalid credentials (counter still incremented to defeat probing)",
			email:    "ghost@example.com",
			password: "anything",
			setup: func(f *fixture) {
				f.redis.On("Incr", mock.Anything, "login_attempts:ghost@example.com").Return(int64(1), nil).Once()
				f.redis.On("Expire", mock.Anything, "login_attempts:ghost@example.com", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.users.On("GetByEmail", mock.Anything, users.GetByEmailRequest{Email: "ghost@example.com"}).
					Return(users.GetByEmailResponse{}, apperror.NotFound("email not found")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
			wantErrMsg:  "invalid email or password",
		},
		{
			name:     "lockout after exceeding LoginMaxAttempts surfaces as Forbidden, no GetByEmail call",
			email:    "victim@example.com",
			password: "Pwd_123!",
			setup: func(f *fixture) {
				// 6th attempt — fixture caps at 5. Lockout fires before
				// the user lookup so the attacker can't even confirm
				// the email exists.
				f.redis.On("Incr", mock.Anything, "login_attempts:victim@example.com").Return(int64(6), nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.Login(context.Background(), auth.LoginRequest{Email: tt.email, Password: tt.password})

			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, "access-tok", out.AccessToken)
				assert.Equal(t, "refresh-tok", out.RefreshToken)
				assert.Equal(t, "user-1", out.User.ID)
				return
			}
			require.Error(t, err)
			var domErr *apperror.DomainError
			require.True(t, errors.As(err, &domErr))
			assert.Equal(t, tt.wantErrType, domErr.Type)
			if tt.wantErrMsg != "" {
				assert.Equal(t, tt.wantErrMsg, domErr.Message)
			}
		})
	}
}
