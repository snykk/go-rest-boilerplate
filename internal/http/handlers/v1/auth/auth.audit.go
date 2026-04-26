package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
)

// auditFromGin builds the HTTP-context portion of an audit Event
// (IP, user-agent, request_id) so call sites only need to fill in
// the event-specific fields. Lives in this package because the auth
// flows are the only ones that audit; the user CRUD endpoints emit
// nothing because they're side-effect-free reads.
func auditFromGin(c *gin.Context) audit.Event {
	requestID := ""
	if v, ok := c.Get("X-Request-ID"); ok {
		if s, ok := v.(string); ok {
			requestID = s
		}
	}
	return audit.Event{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		RequestID: requestID,
	}
}
