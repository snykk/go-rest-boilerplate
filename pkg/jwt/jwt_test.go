package jwt_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, "token is not valid", err.Error())
	})
}
