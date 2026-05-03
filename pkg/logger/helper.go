package logger

import "context"

// GetTraceIDFromContext extracts trace ID from context
func GetTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return traceID
	}

	return ""
}

// GetRequestIDFromContext extracts the external X-Request-ID from
// context. Empty when the request didn't carry one and the middleware
// hasn't run yet.
func GetRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		return requestID
	}
	return ""
}

// ConvertMapToFields converts a map to Fields
func ConvertMapToFields(data map[string]interface{}) Fields {
	fields := make(Fields, len(data))
	for k, v := range data {
		fields[k] = v
	}
	return fields
}

// MergeFields merges multiple Fields into one
func MergeFields(fieldMaps ...Fields) Fields {
	totalSize := 0
	for _, f := range fieldMaps {
		totalSize += len(f)
	}

	result := make(Fields, totalSize)
	for _, f := range fieldMaps {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}
