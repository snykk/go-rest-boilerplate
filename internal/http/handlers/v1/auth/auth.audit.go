package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// auditFromGin builds the HTTP-context portion of an audit Event
// (IP, user-agent, request_id, trace_id) so call sites only need to
// fill in the event-specific fields. The correlation IDs let audit
// entries be joined back to the structured application logs and
// (via trace_id) to spans in the tracing backend.
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
		TraceID:   logger.GetTraceIDFromContext(c.Request.Context()),
	}
}
