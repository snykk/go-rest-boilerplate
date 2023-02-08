package middlewares_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(middlewares.CORSMiddleware())

	router.GET("/test", func(c *gin.Context) {
		fmt.Println("ini request header", c.Request.Header)
		c.String(http.StatusOK, "test")
	})

	t.Run("Test 1 | Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, constants.AllowOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, constants.AllowMethods, w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, constants.AllowHeader, w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
	t.Run("Test 1 | Use Not Allowed Method", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, constants.AllowOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, constants.AllowMethods, w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, constants.AllowHeader, w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
	t.Run("Test 1 | Use Not Allowed Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("misc", "something")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, constants.AllowOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, constants.AllowMethods, w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, constants.AllowHeader, w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
}
