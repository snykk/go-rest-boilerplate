package logger

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
)

func HTTPLogger(param gin.LogFormatterParams) string {
	var color string
	switch {
	case param.StatusCode >= 500:
		color = constants.Red
	case param.StatusCode >= 400:
		color = constants.Yellow
	default:
		color = constants.Green
	}
	return fmt.Sprintf("[LOGGING HTTP] [%s] \033[%sm %d \033[0m %s %s %d %s %s %s\n",
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
