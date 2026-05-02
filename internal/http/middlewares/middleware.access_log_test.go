package middlewares_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/stretchr/testify/assert"
)

func TestAccessLogFormatter_DefaultRequestIDFallback(t *testing.T) {
	params := gin.LogFormatterParams{
		Request:    &http.Request{Header: http.Header{"User-Agent": []string{"test_agent"}}},
		TimeStamp:  time.Now(),
		Method:     "GET",
		Path:       "/test",
		Latency:    100 * time.Millisecond,
		ClientIP:   "127.0.0.1",
		StatusCode: 200,
	}
	got := middlewares.AccessLogFormatter(params)
	assert.Contains(t, got, "req=-")
	assert.Contains(t, got, "GET")
	assert.Contains(t, got, "/test")
}

func TestAccessLogFormatter_WithRequestID(t *testing.T) {
	params := gin.LogFormatterParams{
		Request:    &http.Request{Header: http.Header{"User-Agent": []string{"test_agent"}}},
		TimeStamp:  time.Now(),
		Method:     "GET",
		Path:       "/test",
		Latency:    100 * time.Millisecond,
		ClientIP:   "127.0.0.1",
		StatusCode: 500,
		Keys:       map[any]any{"X-Request-ID": "abc-123"},
	}
	got := middlewares.AccessLogFormatter(params)
	assert.Contains(t, got, "req=abc-123")
	assert.True(t, strings.Contains(got, "\033[41m"), "5xx must render with red color code")
}
