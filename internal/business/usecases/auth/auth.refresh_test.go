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

func refreshClaims(jti, email string) jwt.JwtCustomClaim {
	return jwt.JwtCustomClaim{
		UserID: "user-1",
		Email:  email,
		Kind:   jwt.KindRefresh,
		RegisteredClaims: golangJWT.RegisteredClaims{ID: jti},
	}
}

func TestRefresh_HappyPath_RotatesAndRevokes(t *testing.T) {
	f := newFixture(t)
	user := activeUser(t)
	oldJTI := "old-jti"

	f.jwt.On("ParseRefreshToken", "old-refresh-tok").Return(refreshClaims(oldJTI, user.Email), nil).Once()
	f.redis.On("Get", mock.Anything, "refresh:"+oldJTI).Return(oldJTI, nil).Once()
	f.users.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
	f.jwt.On("GenerateTokenPair", user.ID, false, user.Email).Return(samplePair(), nil).Once()
	f.redis.On("Set", mock.Anything, "refresh:refresh-jti", "refresh-jti").Return(nil).Once()
	f.redis.On("Expire", mock.Anything, "refresh:refresh-jti", mock.AnythingOfType("time.Duration")).Return(nil).Once()
	// Old jti deleted last (after the new one is persisted).
	f.redis.On("Del", mock.Anything, "refresh:"+oldJTI).Return(nil).Once()

	out, err := f.usecase.Refresh(context.Background(), "old-refresh-tok")
	require.NoError(t, err)
	assert.Equal(t, "access-tok", out.AccessToken)
	assert.Equal(t, "refresh-tok", out.RefreshToken)
}

func TestRefresh_RevokedToken(t *testing.T) {
	f := newFixture(t)
	jti := "stale-jti"

	f.jwt.On("ParseRefreshToken", "stale-tok").Return(refreshClaims(jti, "x@y.com"), nil).Once()
	// Redis Get returns an error → token has been revoked.
	f.redis.On("Get", mock.Anything, "refresh:"+jti).Return("", errors.New("redis: nil")).Once()

	_, err := f.usecase.Refresh(context.Background(), "stale-tok")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}

func TestRefresh_InvalidTokenSignature(t *testing.T) {
	f := newFixture(t)
	f.jwt.On("ParseRefreshToken", "bogus").Return(jwt.JwtCustomClaim{}, errors.New("bad signature")).Once()

	_, err := f.usecase.Refresh(context.Background(), "bogus")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}
