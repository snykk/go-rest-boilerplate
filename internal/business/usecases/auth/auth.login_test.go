package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
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
		// extraAsserts runs only on the happy path.
		extraAsserts func(t *testing.T, f *fixture, out any)
	}{
		{
			name:     "happy path issues token pair and persists refresh JTI",
			email:    "patrick@example.com",
			password: "Pwd_123!",
			setup: func(f *fixture) {
				user := activeUser(t)
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
				f.jwt.On("GenerateTokenPair", user.ID, false, user.Email).Return(samplePair(), nil).Once()
				f.redis.On("Set", mock.Anything, "refresh:refresh-jti", "refresh-jti").Return(nil).Once()
				f.redis.On("Expire", mock.Anything, "refresh:refresh-jti", mock.AnythingOfType("time.Duration")).Return(nil).Once()
			},
		},
		{
			name:     "wrong password returns Unauthorized with generic message",
			email:    "patrick@example.com",
			password: "wrong-password",
			setup: func(f *fixture) {
				// No JWT calls expected — password check fails first.
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(activeUser(t), nil).Once()
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
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeForbidden,
		},
		{
			name:     "unknown email surfaces as invalid credentials (not 404)",
			email:    "ghost@example.com",
			password: "anything",
			setup: func(f *fixture) {
				// No JWT, no Redis — Unauthorized returns before mint.
				// Same wantErrMsg as wrong-password to verify the
				// timing-attack mitigation: an attacker can't enumerate
				// registered emails by reading the error string.
				f.users.On("GetByEmail", mock.Anything, "ghost@example.com").
					Return(domain.User{}, apperror.NotFound("email not found")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
			wantErrMsg:  "invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.Login(context.Background(), tt.email, tt.password)

			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, "access-tok", out.AccessToken)
				assert.Equal(t, "refresh-tok", out.RefreshToken)
				assert.Equal(t, "user-1", out.User.ID)
				if tt.extraAsserts != nil {
					tt.extraAsserts(t, f, out)
				}
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
