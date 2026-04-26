package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Common body-size ceilings. Routes apply the tightest one that
// makes sense for the payload they accept. The global default
// (DefaultBodyMaxBytes) is the last line of defense for any route
// that doesn't set its own.
const (
	// DefaultBodyMaxBytes is the catch-all global ceiling — 1 MiB.
	DefaultBodyMaxBytes int64 = 1 << 20

	// AuthBodyMaxBytes covers register / login / OTP / refresh /
	// logout payloads. None of these carry more than a few hundred
	// bytes of JSON; capping at 4 KiB blocks slow-body attacks
	// without rejecting legitimate traffic.
	AuthBodyMaxBytes int64 = 4 << 10
)

// BodySizeLimitMiddleware caps the request body via http.MaxBytesReader.
// Stacking multiple instances narrows the ceiling: the innermost
// (most-recently-applied) wrapper wins, so route-scoped limits below
// the global one do tighten the actual cap.
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
