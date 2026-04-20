package logger

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// color
const (
	Red    = "41"
	Yellow = "43"
	Green  = "42"
)

// requestIDContextKey mirrors middlewares.RequestIDHeader without
// creating a package-level import cycle (logger is imported by more
// packages than middlewares is).
const requestIDContextKey = "X-Request-ID"

func HTTPLogger(param gin.LogFormatterParams) string {
	var color string
	switch {
	case param.StatusCode >= 500:
		color = Red
	case param.StatusCode >= 400:
		color = Yellow
	default:
		color = Green
	}

	requestID := "-"
	if v, ok := param.Keys[requestIDContextKey]; ok {
		if s, ok := v.(string); ok && s != "" {
			requestID = s
		}
	}

	return fmt.Sprintf("[LOGGING HTTP] [%s] req=%s \033[%sm %d \033[0m %s %s %s %s %s %s\n",
		param.TimeStamp.Format("2006-01-02 15:04:05"),
		requestID,
		color,
		param.StatusCode,
		param.Method,
		param.Path,
		param.Latency,
		param.ClientIP,
		param.ErrorMessage,
		param.Request.UserAgent(),
	)
}
