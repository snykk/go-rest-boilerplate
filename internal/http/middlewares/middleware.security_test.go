package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/stretchr/testify/assert"
)

func newSecRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middlewares.SecurityHeadersMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func TestSecurityHeaders_DevDefaults(t *testing.T) {
	config.AppConfig.Environment = constants.EnvironmentDevelopment
	r := newSecRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
	// HSTS must NOT be sent in development — would teach localhost
	// browsers to refuse plain HTTP.
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_ProductionAddsHSTS(t *testing.T) {
	config.AppConfig.Environment = constants.EnvironmentProduction
	t.Cleanup(func() { config.AppConfig.Environment = constants.EnvironmentDevelopment })

	r := newSecRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=")
	assert.Contains(t, hsts, "includeSubDomains")
}
