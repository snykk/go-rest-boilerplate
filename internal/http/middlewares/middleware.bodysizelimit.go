package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodySizeLimitMiddleware limits the request body size to the given number of bytes.
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
