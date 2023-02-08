package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	jwtService := jwt.NewJWTService(config.AppConfig.JWTSecret, config.AppConfig.JWTIssuer, config.AppConfig.JWTExpired)
	token, err := jwtService.GenerateToken("asf-asf-asfdasd-asdfsa", false, "john.doe@example.com", "password")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestParseToken(t *testing.T) {
	t.Run("With Valid Token", func(t *testing.T) {
		jwtService := jwt.NewJWTService(config.AppConfig.JWTSecret, config.AppConfig.JWTIssuer, config.AppConfig.JWTExpired)
		config.AppConfig.JWTExpired = 5

		token, _ := jwtService.GenerateToken("asf-asf-asfdasd-asdfsa", false, "john.doe@example.com", "password")

		claims, err := jwtService.ParseToken(token)
		fmt.Println("ini expire token", claims.StandardClaims.ExpiresAt)
		assert.NoError(t, err)
		assert.Equal(t, "asf-asf-asfdasd-asdfsa", claims.UserID)
		assert.Equal(t, false, claims.IsAdmin)
		assert.Equal(t, "john.doe@example.com", claims.Email)
		assert.Equal(t, "password", claims.Password)
		assert.True(t, claims.StandardClaims.ExpiresAt >= time.Now().Unix())
		assert.Equal(t, config.AppConfig.JWTIssuer, claims.StandardClaims.Issuer)
		assert.True(t, claims.StandardClaims.IssuedAt <= time.Now().Unix())
	})
	t.Run("With Invalid Token", func(t *testing.T) {
		jwtService := jwt.NewJWTService(config.AppConfig.JWTSecret, config.AppConfig.JWTIssuer, config.AppConfig.JWTExpired)

		_, err := jwtService.ParseToken("invalid_token")
		assert.Error(t, err)
		assert.Equal(t, "token is not valid", err.Error())
	})
}
