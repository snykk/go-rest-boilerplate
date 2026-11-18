package httpserver

import (
	"log/slog"
	"net/http"
	"net/netip"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/rs/cors"
	"go.opentelemetry.io/otel/trace"

	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
)

const (
	apiCSP     = "default-src 'none'; frame-ancestors 'none'"
	swaggerCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; frame-ancestors 'none'"
)

// Middleware wraps a handler with behaviour that runs around it.
type Middleware func(http.Handler) http.Handler

// Chain wraps handler with middlewares. middlewares[0] is the outermost and sees each request first.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range slices.Backward(middlewares) {
		handler = middleware(handler)
	}
	return handler
}

// RequestID keeps an incoming X-Request-ID that requestid.Valid accepts. It replaces a missing or invalid one
// with the trace ID of the request's span, so one ID finds the request in logs and traces, or with a new
// UUIDv7 when the request has no trace. Requests that continue one caller's trace without an X-Request-ID of
// their own therefore share an ID; their span IDs tell them apart. It sets the ID on the response header and on
// the context of the request passed on.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestid.Header)
			if !requestid.Valid(requestID) {
				requestID = newRequestID(r)
			}

			w.Header().Set(requestid.Header, requestID)
			next.ServeHTTP(w, r.WithContext(requestid.With(r.Context(), requestID)))
		})
	}
}

// AccessLog writes one "http request" record after each request with method, route (the ServeMux pattern,
// or "unmatched"), status, bytes, duration_ms and client_ip (see ClientIP). Requests to the paths in quiet,
// such as health probes, are logged at DEBUG; of the others, responses with status 500 or more are logged at
// ERROR and the rest at INFO. The route is read from the request it passes on, so the handlers between it and
// the ServeMux must pass that same request.
func AccessLog(log *slog.Logger, trusted []netip.Prefix, quiet ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			response := &recorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(response, r)

			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			level := slog.LevelInfo
			switch {
			case slices.Contains(quiet, r.URL.Path):
				level = slog.LevelDebug
			case response.status >= http.StatusInternalServerError:
				level = slog.LevelError
			}

			log.LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", response.status),
				slog.Int64("bytes", response.bytes),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("client_ip", ClientIP(r, trusted)),
			)
		})
	}
}

// Recover turns a panic in a later handler into a 500 internal problem and logs the panic value and stack
// at ERROR; the response never carries the panic value. It re-panics http.ErrAbortHandler, which aborts a
// response on purpose.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if recovered == http.ErrAbortHandler {
					panic(recovered) // net/http expects it back: it aborts the response without logging a stack
				}

				log.LogAttrs(r.Context(), slog.LevelError, "panic",
					slog.Any("panic", recovered), slog.String("stack", string(debug.Stack())))
				WriteProblem(w, r, Problem{
					Status: http.StatusInternalServerError,
					Code:   "internal",
					Detail: "internal server error",
				})
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders sets headers that stop browsers from sniffing, framing or loading anything the API does
// not serve. Under /swagger/ the Content-Security-Policy lets the Swagger UI load its own scripts, styles
// and images. With production set it also sends Strict-Transport-Security.
func SecurityHeaders(production bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", "DENY")
			header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			header.Set("Permissions-Policy",
				"accelerometer=(), camera=(), geolocation=(), gyroscope=(), microphone=(), payment=(), usb=()")

			csp := apiCSP
			if strings.HasPrefix(r.URL.Path, "/swagger/") {
				csp = swaggerCSP
			}
			header.Set("Content-Security-Policy", csp)

			if production {
				header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS answers preflight requests and adds CORS headers for the listed origins; "*" allows every origin.
// Browsers may send Content-Type, Authorization and X-Request-ID, and may read X-Request-ID and Location.
// Credentials are never allowed.
func CORS(origins []string) Middleware {
	policy := cors.New(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowedHeaders: []string{"Content-Type", "Authorization", requestid.Header},
		ExposedHeaders: []string{requestid.Header, "Location"},
		MaxAge:         600,
	})
	return policy.Handler
}

// BodyLimit caps each request body at maxBytes. Reading past the cap fails with *http.MaxBytesError, which
// DecodeJSON reports as a 413 problem.
func BodyLimit(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func newRequestID(r *http.Request) string {
	if spanContext := trace.SpanContextFromContext(r.Context()); spanContext.HasTraceID() {
		return spanContext.TraceID().String()
	}
	return id.New()
}

type recorder struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
}

func (r *recorder) WriteHeader(status int) {
	if !r.wroteHeader && status >= http.StatusOK {
		r.status, r.wroteHeader = status, true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *recorder) Write(data []byte) (int, error) {
	r.wroteHeader = true
	written, err := r.ResponseWriter.Write(data)
	r.bytes += int64(written)
	return written, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
