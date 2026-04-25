package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
)

// SecurityHeadersMiddleware sets a small but high-leverage set of
// browser-side security headers on every response. The API itself
// doesn't render HTML, but credentialed XHR / fetch calls from a
// browser still benefit, and these headers are cheap insurance against
// future endpoints that *do* serve HTML (admin panels, email previews,
// etc.) where a missing header would be a real exposure.
//
//	X-Content-Type-Options: nosniff             — disables MIME sniffing
//	X-Frame-Options:        DENY                — blocks clickjacking via <iframe>
//	Referrer-Policy:        strict-origin-...   — caps the data leaked in Referer
//	Content-Security-Policy: default-src 'none' — APIs return JSON, never need to load anything
//	Strict-Transport-Security                    — production only, requires real HTTPS
func SecurityHeadersMiddleware() gin.HandlerFunc {
	isProduction := config.AppConfig.Environment == constants.EnvironmentProduction
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// API responses are JSON; they never legitimately load
		// scripts, styles, frames, or images. default-src 'none' makes
		// any accidental HTML response inert in the browser.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// Permissions-Policy: deny everything by default. Cheap and
		// covers powerful APIs (camera, geolocation, etc.) that an
		// API surface should never need.
		h.Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
		if isProduction {
			// HSTS only in production — sending it from a dev server
			// on http://localhost teaches the browser to refuse plain
			// HTTP for that host for a year, which is a footgun.
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
