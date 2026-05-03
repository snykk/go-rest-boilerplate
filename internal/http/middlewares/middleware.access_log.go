package middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// access-log color codes (xterm SGR background).
const (
	accessLogRed    = "41"
	accessLogYellow = "43"
	accessLogGreen  = "42"
)

// requestIDContextKey mirrors RequestIDHeader without introducing a
// circular import (this file is in the same package as the request-id
// middleware, but we keep the constant local for clarity in tests).
const requestIDContextKey = "X-Request-ID"

// AccessLogFormatter renders a one-line access log per request,
// suitable for gin.LoggerWithFormatter. Status code is colorized so
// 5xx / 4xx jump out in plain `tail -f` sessions.
func AccessLogFormatter(param gin.LogFormatterParams) string {
	var color string
	switch {
	case param.StatusCode >= 500:
		color = accessLogRed
	case param.StatusCode >= 400:
		color = accessLogYellow
	default:
		color = accessLogGreen
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
