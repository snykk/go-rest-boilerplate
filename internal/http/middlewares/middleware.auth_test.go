package middlewares_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

var (
	jwtService          jwt.JWTService
	s                   *gin.Engine
	authBasicMiddleware gin.HandlerFunc
	authAdminMiddleware gin.HandlerFunc
)

const (
	adminEndpoint = "/admin"
	forEveryone   = "/everyone"
)

func authenticatedHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":  true,
		"message": "nice to meet you again sir...",
	})
}

func setup(t *testing.T) {
	jwtService = jwt.NewJWTService(config.AppConfig.JWTSecret, config.AppConfig.JWTIssuer, config.AppConfig.JWTExpired)
	authBasicMiddleware = middlewares.NewAuthMiddleware(jwtService, false)
	authAdminMiddleware = middlewares.NewAuthMiddleware(jwtService, true)

	s = gin.New()
	s.GET(forEveryone, authBasicMiddleware, authenticatedHandler)
	s.GET(adminEndpoint, authAdminMiddleware, authenticatedHandler)
}

func generateToken(isAdmin bool) (token string, err error) {
	token, err = jwtService.GenerateToken("ddfcea5c-d919-4a8f-a631-4ace39337s3a", isAdmin, "najibfikri13@gmail.com", "12345678")
	return
}

func getAdminToken() (string, error) {
	return generateToken(true)
}

func getBasicToken() (string, error) {
	return generateToken(false)
}

func TestAuthMiddleware(t *testing.T) {
	setup(t)
	// Define route

	t.Run("Test 1 | Success Get Admin Handler", func(t *testing.T) {
		token, err := getAdminToken()
		if err != nil {
			t.Error(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, adminEndpoint, nil)

		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "nice to meet you again sir")
	})
	t.Run("Test 2 | Invalid Token", func(t *testing.T) {
		token := "mwehehe"

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, forEveryone, nil)

		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()
		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "invalid token")
	})
	t.Run("Test 3 | Must Content Bearer", func(t *testing.T) {
		token, err := getBasicToken()
		if err != nil {
			t.Error(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, forEveryone, nil)

		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", fmt.Sprintf("Token %s", token))

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()
		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "token must content bearer")
	})
	t.Run("Test 4 | Invalid Format", func(t *testing.T) {
		token, err := getBasicToken()
		if err != nil {
			t.Error(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, forEveryone, nil)

		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", fmt.Sprintf("Bearer token: %s", token))

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()
		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "invalid header format")
	})
	t.Run("Test 4 | Not Authorize", func(t *testing.T) {
		token, err := getBasicToken()
		if err != nil {
			t.Error(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, adminEndpoint, nil)

		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()
		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "you don't have access for this action")
	})
}
