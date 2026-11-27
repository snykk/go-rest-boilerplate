// Package logger builds the service's slog logger.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/trace"

	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
)

// Redacted is what a secret's value is written as.
const Redacted = "[REDACTED]"

var secretWords = []string{"password", "passwd", "secret", "token", "authorization", "cookie", "api_key",
	"apikey", "private_key"}

// Options configure New.
type Options struct {
	// Level is the least severe level written. A *slog.LevelVar lets it change while the service runs.
	Level slog.Leveler
	// Format is "text" or "json".
	Format string
	// Source adds the attribute source: the file and line of the logging call, relative to the module root.
	Source bool
	// Attrs are added to every record, such as the service name, its version and the environment.
	Attrs []slog.Attr
}

// New returns a logger that writes records at options.Level and above to output. Every record carries options.Attrs,
// the request_id, trace_id and span_id of its context when it has them, and source when options.Source is set. The
// value of every attribute whose key contains password, passwd, secret, token, authorization, cookie, api_key, apikey
// or private_key, in any case and in any group, is written as Redacted. New returns an error for a format
// other than "text" or "json".
func New(output io.Writer, options Options) (*slog.Logger, error) {
	handlerOptions := &slog.HandlerOptions{Level: options.Level, ReplaceAttr: redact}
	var handler slog.Handler
	switch options.Format {
	case "text":
		handler = slog.NewTextHandler(output, handlerOptions)
	case "json":
		handler = slog.NewJSONHandler(output, handlerOptions)
	default:
		return nil, fmt.Errorf("unknown log format %q: want text or json", options.Format)
	}

	if len(options.Attrs) > 0 {
		handler = handler.WithAttrs(options.Attrs)
	}
	return slog.New(contextHandler{Handler: handler, source: options.Source}), nil
}

type contextHandler struct {
	slog.Handler
	source bool
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if id := requestid.From(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		record.AddAttrs(slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()))
	}
	if h.source && record.PC != 0 {
		record.AddAttrs(slog.String("source", sourceOf(record.PC)))
	}
	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attributes), source: h.source}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name), source: h.source}
}

func redact(_ []string, attribute slog.Attr) slog.Attr {
	key := strings.ToLower(attribute.Key)
	for _, word := range secretWords {
		if strings.Contains(key, word) {
			return slog.String(attribute.Key, Redacted)
		}
	}
	return attribute
}

func sourceOf(programCounter uintptr) string {
	frame, _ := runtime.CallersFrames([]uintptr{programCounter}).Next()
	return strings.TrimPrefix(frame.File, moduleRoot()) + ":" + strconv.Itoa(frame.Line)
}

// moduleRoot is the path prefix of the module's files as the binary records them: a directory, or the module
// path when the binary is built with -trimpath.
var moduleRoot = sync.OnceValue(func() string {
	_, file, _, _ := runtime.Caller(0)
	return strings.TrimSuffix(file, "internal/platform/logger/logger.go")
})
