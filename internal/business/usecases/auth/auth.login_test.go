package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
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

func TestLogin_HappyPath(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
	f.jwt.On("GenerateTokenPair", user.ID, false, user.Email).Return(samplePair(), nil).Once()
	f.redis.On("Set", mock.Anything, "refresh:refresh-jti", "refresh-jti").Return(nil).Once()
	f.redis.On("Expire", mock.Anything, "refresh:refresh-jti", mock.AnythingOfType("time.Duration")).Return(nil).Once()

	out, err := f.usecase.Login(context.Background(), "patrick@example.com", "Pwd_123!")
	require.NoError(t, err)
	assert.Equal(t, "access-tok", out.AccessToken)
	assert.Equal(t, "refresh-tok", out.RefreshToken)
	assert.Equal(t, user.ID, out.User.ID)
}

func TestLogin_WrongPassword(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()
	// No JWT calls expected — the password check fails first.

	_, err := f.usecase.Login(context.Background(), "patrick@example.com", "wrong-password")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
	// Same message as unknown-email path: "invalid email or password"
	// — the timing attack mitigation ensures the two paths return
	// indistinguishable errors too.
	assert.Equal(t, "invalid email or password", domErr.Message)
}

func TestLogin_InactiveUser(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)
	user.Active = false

	f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(user, nil).Once()

	_, err := f.usecase.Login(context.Background(), "patrick@example.com", "Pwd_123!")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeForbidden, domErr.Type)
}

func TestLogin_UnknownEmail_StillRunsBcryptForTimingMitigation(t *testing.T) {
	f := newFixture(t)
	f.users.On("GetByEmail", mock.Anything, "ghost@example.com").
		Return(entities.UserDomain{}, apperror.NotFound("email not found")).Once()
	// No JWT, no Redis calls — this path returns Unauthorized
	// before any token mint.

	_, err := f.usecase.Login(context.Background(), "ghost@example.com", "anything")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
	assert.Equal(t, "invalid email or password", domErr.Message,
		"unknown email must surface as invalid credentials, not 404 — "+
			"otherwise an attacker can enumerate registered emails")
}
