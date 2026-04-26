package jwt_test

import (
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/pkg/clock"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret  = "test-secret-key"
	testIssuer  = "test-issuer"
	testExpired = 5
)

func TestGenerateToken(t *testing.T) {
	jwtService := jwt.NewJWTService(testSecret, testIssuer, testExpired)
	token, err := jwtService.GenerateToken("asf-asf-asfdasd-asdfsa", false, "john.doe@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestParseToken(t *testing.T) {
	t.Run("With Valid Token", func(t *testing.T) {
		jwtService := jwt.NewJWTService(testSecret, testIssuer, testExpired)

		token, _ := jwtService.GenerateToken("asf-asf-asfdasd-asdfsa", false, "john.doe@example.com")

		claims, err := jwtService.ParseToken(token)
		assert.NoError(t, err)
		assert.Equal(t, "asf-asf-asfdasd-asdfsa", claims.UserID)
		assert.Equal(t, false, claims.IsAdmin)
		assert.Equal(t, "john.doe@example.com", claims.Email)
		assert.True(t, claims.ExpiresAt.Time.After(time.Now()) || claims.ExpiresAt.Time.Equal(time.Now()))
		assert.Equal(t, testIssuer, claims.Issuer)
		assert.True(t, claims.IssuedAt.Time.Before(time.Now()) || claims.IssuedAt.Time.Equal(time.Now()))
	})
	t.Run("With Invalid Token", func(t *testing.T) {
		jwtService := jwt.NewJWTService(testSecret, testIssuer, testExpired)

		_, err := jwtService.ParseToken("invalid_token")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, jwt.ErrInvalidToken), "expected error to wrap ErrInvalidToken, got %v", err)
	})
}

func TestGenerateTokenPair_RespectsInjectedClock(t *testing.T) {
	// Freeze time at a known instant so the asserted ExpiresAt and
	// IssuedAt values are exact — no flake from time.Now() drift.
	at := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	svc := jwt.WithClock(
		jwt.NewJWTServiceWithRefresh(testSecret, testIssuer, 5, 7),
		clock.Frozen(at),
	)

	pair, err := svc.GenerateTokenPair("user-1", false, "alice@example.com")
	require.NoError(t, err)

	// Access token: IssuedAt = at, ExpiresAt = at + 5h.
	accessClaims, err := svc.ParseToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, at, accessClaims.IssuedAt.Time.UTC())
	assert.Equal(t, at.Add(5*time.Hour), accessClaims.ExpiresAt.Time.UTC())

	// Refresh token: IssuedAt = at, ExpiresAt = at + 7d.
	refreshClaims, err := svc.ParseRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, at, refreshClaims.IssuedAt.Time.UTC())
	assert.Equal(t, at.Add(7*24*time.Hour), refreshClaims.ExpiresAt.Time.UTC())
}
