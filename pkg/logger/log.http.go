package logger

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	red    = "41"
	yellow = "43"
	green  = "42"
)

func CustomLogFormatter(param gin.LogFormatterParams) string {
	var color string
	switch {
	case param.StatusCode >= 500:
		color = red
	case param.StatusCode >= 400:
		color = yellow
	default:
		color = green
	}
	return fmt.Sprintf("[LOGGER] [%s] \033[%sm %d \033[0m %s %s %d %s %s %s\n",
		param.TimeStamp.Format("2006-01-02 15:04:05"),
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
