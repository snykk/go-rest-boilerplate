package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
)

// jsonOptions are Options for JSON at Info, changed by change.
func jsonOptions(change func(options *logger.Options)) logger.Options {
	options := logger.Options{
		Level:  slog.LevelInfo,
		Format: "json",
	}
	if change != nil {
		change(&options)
	}
	return options
}

// tracedContext carries a sampled span with fixed trace and span IDs.
func tracedContext() context.Context {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x0a, 0xf7, 0x65, 0x19, 0x16, 0xcd, 0x43, 0xdd, 0x84, 0x48, 0xeb, 0x21, 0x1c, 0x80, 0x31, 0x9c},
		SpanID:     trace.SpanID{0xb7, 0xad, 0x6b, 0x71, 0x69, 0x20, 0x33, 0x31},
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), spanContext)
}

// New writes records at the chosen level in the chosen format. Every record carries the given attributes,
// the request, trace and span IDs of its context, and, when asked, the file and line of the logging call. The
// values of keys that name secrets are redacted.
func TestNew(t *testing.T) {
	debugLevel := new(slog.LevelVar)
	debugLevel.Set(slog.LevelDebug)
	tCases := []struct {
		name         string
		opts         logger.Options
		ctx          context.Context
		log          func(ctx context.Context, log *slog.Logger) // nil logs "hello" at Info
		wantErr      bool
		wantEmpty    bool           // nothing is written
		wantJSON     map[string]any // fields the JSON line carries
		wantSource   string         // the prefix the source field starts with
		wantNoFields []string       // fields the JSON line must not carry
		wantAbsent   []string       // text the output must not contain
	}{
		{
			name:     "a record carries the request ID of its context",
			opts:     jsonOptions(nil),
			ctx:      requestid.With(context.Background(), "rid-1"),
			wantJSON: map[string]any{"msg": "hello", "request_id": "rid-1"},
		},
		{
			name: "a record carries the trace and span IDs of its context",
			opts: jsonOptions(nil),
			ctx:  tracedContext(),
			wantJSON: map[string]any{
				"trace_id": "0af7651916cd43dd8448eb211c80319c",
				"span_id":  "b7ad6b7169203331",
			},
		},
		{
			name:         "a context without IDs adds none",
			opts:         jsonOptions(nil),
			ctx:          context.Background(),
			wantNoFields: []string{"request_id", "trace_id", "span_id"},
		},
		{
			name: "the given attributes are on every record",
			opts: jsonOptions(func(options *logger.Options) {
				options.Attrs = []slog.Attr{slog.String("service", "articles"), slog.String("version", "v1.2.3")}
			}),
			ctx:      context.Background(),
			wantJSON: map[string]any{"service": "articles", "version": "v1.2.3"},
		},
		{
			name:       "the source is the file and line of the logging call, relative to the module root",
			opts:       jsonOptions(func(options *logger.Options) { options.Source = true }),
			ctx:        context.Background(),
			wantSource: "internal/platform/logger/logger_test.go:",
		},
		{
			name:         "without Source no source is written",
			opts:         jsonOptions(nil),
			ctx:          context.Background(),
			wantNoFields: []string{"source"},
		},
		{
			name: "the values of keys that name secrets are redacted, in groups too",
			opts: jsonOptions(nil),
			ctx:  context.Background(),
			log: func(ctx context.Context, log *slog.Logger) {
				log.InfoContext(ctx, "login", "user", "alice", "password", "hunter2", "Authorization", "Bearer abc",
					slog.Group("db", "db_password", "s3cret", "host", "pg"))
			},
			wantJSON: map[string]any{
				"user":          "alice",
				"password":      "[REDACTED]",
				"Authorization": "[REDACTED]",
			},
			wantAbsent: []string{"hunter2", "Bearer abc", "s3cret"},
		},
		{
			name:      "a record below the level is not written",
			opts:      jsonOptions(nil),
			ctx:       context.Background(),
			log:       func(ctx context.Context, log *slog.Logger) { log.DebugContext(ctx, "hello") },
			wantEmpty: true,
		},
		{
			name:     "a level variable sets the level",
			opts:     jsonOptions(func(options *logger.Options) { options.Level = debugLevel }),
			ctx:      context.Background(),
			log:      func(ctx context.Context, log *slog.Logger) { log.DebugContext(ctx, "hello") },
			wantJSON: map[string]any{"msg": "hello", "level": "DEBUG"},
		},
		{name: "an unknown format is refused", opts: logger.Options{Format: "xml"}, wantErr: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			log, err := logger.New(&logs, tc.opts)
			if (err != nil) != tc.wantErr {
				t.Fatalf("New = %v, want an error: %t", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if tc.log != nil {
				tc.log(tc.ctx, log)
			} else {
				log.InfoContext(tc.ctx, "hello")
			}

			logged := logs.String()
			if tc.wantEmpty {
				if logged != "" {
					t.Errorf("output = %q, want nothing", logged)
				}
				return
			}
			for _, text := range tc.wantAbsent {
				if strings.Contains(logged, text) {
					t.Errorf("output = %q, want no %q", logged, text)
				}
			}
			var line map[string]any
			if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
				t.Fatalf("output %q is not JSON: %v", logged, err)
			}
			for key, value := range tc.wantJSON {
				if line[key] != value {
					t.Errorf("%s = %v, want %v", key, line[key], value)
				}
			}
			for _, key := range tc.wantNoFields {
				if _, ok := line[key]; ok {
					t.Errorf("%s = %v, want no such field", key, line[key])
				}
			}
			if source, _ := line["source"].(string); tc.wantSource != "" && !strings.HasPrefix(source, tc.wantSource) {
				t.Errorf("source = %q, want it to start with %q", source, tc.wantSource)
			}
		})
	}
}
