package auth_test

import (
	"context"
	"errors"
	"testing"

	golangJWT "github.com/golang-jwt/jwt/v5"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLogout_DeletesRefreshJTI(t *testing.T) {
	f := newFixture(t)
	claims := jwt.JwtCustomClaim{
		Kind:             jwt.KindRefresh,
		RegisteredClaims: golangJWT.RegisteredClaims{ID: "jti-to-delete"},
	}
	f.jwt.On("ParseRefreshToken", "good-tok").Return(claims, nil).Once()
	f.redis.On("Del", mock.Anything, "refresh:jti-to-delete").Return(nil).Once()

	require.NoError(t, f.usecase.Logout(context.Background(), "good-tok"))
}

func TestLogout_InvalidToken(t *testing.T) {
	f := newFixture(t)
	f.jwt.On("ParseRefreshToken", "bad").Return(jwt.JwtCustomClaim{}, errors.New("bad sig")).Once()

	err := f.usecase.Logout(context.Background(), "bad")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}
