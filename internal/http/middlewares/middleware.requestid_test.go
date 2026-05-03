package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRequestIDMiddleware_BridgesIDsToContext(t *testing.T) {
	r := gin.New()
	r.Use(middlewares.RequestIDMiddleware())

	var seenRequestID, seenTraceID string
	r.GET("/probe", func(c *gin.Context) {
		ctx := c.Request.Context()
		seenRequestID = logger.GetRequestIDFromContext(ctx)
		seenTraceID = logger.GetTraceIDFromContext(ctx)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/probe", http.NoBody)
	req.Header.Set("X-Request-ID", "abc-from-client")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "abc-from-client", seenRequestID,
		"request_id from client header must be visible to handlers via logger.GetRequestIDFromContext")
	assert.Empty(t, seenTraceID,
		"trace_id stays empty when otelgin isn't mounted; populated end-to-end in production")
	assert.Equal(t, "abc-from-client", w.Header().Get("X-Request-ID"),
		"request_id must be echoed in the response header")
}

func TestRequestIDMiddleware_GeneratesWhenAbsent(t *testing.T) {
	r := gin.New()
	r.Use(middlewares.RequestIDMiddleware())

	var seen string
	r.GET("/probe", func(c *gin.Context) {
		seen = logger.GetRequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/probe", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, seen, "middleware must generate a UUID when no header is present")
	assert.Equal(t, seen, w.Header().Get("X-Request-ID"))
}
