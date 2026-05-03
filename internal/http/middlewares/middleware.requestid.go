package middlewares

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware accepts an incoming X-Request-ID (or generates a
// UUID if absent), echoes it on the response, and bridges both
// correlation IDs into the request context so logger.*WithContext
// emits them on every log line:
//
//   - request_id: external client-facing ID (W3C-flavored convention).
//     Stays the same end-to-end for the client, even across services.
//   - traceId: OTel-generated W3C trace ID (auto-populated by
//     otelgin upstream of this middleware). Used to link logs to spans
//     in the tracing backend (Jaeger / Tempo / etc.).
//
// Mount this AFTER otelgin so the OTel span context is already
// established when we reach in to extract the trace ID.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDHeader, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)

		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)
		if span := trace.SpanFromContext(ctx); span.SpanContext().HasTraceID() {
			ctx = context.WithValue(ctx, logger.TraceIDKey, span.SpanContext().TraceID().String())
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
