package httpserver_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

// newJSONLogger is a JSON logger writing records at Debug and above into logs.
func newJSONLogger(t *testing.T, logs *bytes.Buffer) *slog.Logger {
	t.Helper()
	log, err := logger.New(logs, logger.Options{
		Level:  slog.LevelDebug,
		Format: "json",
	})
	if err != nil {
		t.Fatalf("logger.New = %v", err)
	}
	return log
}

// Chain applies the first middleware outermost.
func TestChain(t *testing.T) {
	var order []string
	middleware := func(name string) httpserver.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	final := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { order = append(order, "handler") })

	httpserver.Chain(final, middleware("a"), middleware("b")).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if want := []string{"a", "b", "handler"}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

// RequestID keeps a valid incoming X-Request-ID and replaces anything else with a UUIDv7. It puts the
// result in the response header and the request context.
func TestRequestID(t *testing.T) {
	traceID := trace.TraceID{0x0a, 0xf7, 0x65, 0x19, 0x16, 0xcd, 0x43, 0xdd, 0x84, 0x48, 0xeb, 0x21, 0x1c, 0x80, 0x31, 0x9c}
	tCases := []struct {
		name     string
		incoming string
		inTrace  bool // the request's context carries a span of trace traceID
		keep     bool
		want     string // the ID to expect; "" means a new UUIDv7
	}{
		{name: "a request without an ID gets a new one"},
		{
			name:    "a request without an ID in a trace gets the trace ID",
			inTrace: true,
			want:    "0af7651916cd43dd8448eb211c80319c",
		},
		{name: "a valid incoming ID is kept, in a trace too", incoming: "abc-123", inTrace: true, keep: true},
		{name: "a valid incoming ID is kept", incoming: "abc-123", keep: true},
		{name: "an ID over 64 characters is replaced", incoming: strings.Repeat("a", 65)},
		{name: "an ID with a space is replaced", incoming: "a b"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			handler := httpserver.RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = requestid.From(r.Context())
			}))
			request := httptest.NewRequest("GET", "/", nil)
			if tc.incoming != "" {
				request.Header.Set("X-Request-ID", tc.incoming)
			}
			if tc.inTrace {
				spanContext := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     trace.SpanID{1},
					TraceFlags: trace.FlagsSampled,
				})
				request = request.WithContext(trace.ContextWithSpanContext(request.Context(), spanContext))
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			got := recorder.Header().Get("X-Request-ID")
			if got != seen {
				t.Errorf("header ID = %q, context ID = %q, want them equal", got, seen)
			}
			if tc.keep {
				if got != tc.incoming {
					t.Errorf("ID = %q, want %q kept", got, tc.incoming)
				}
				return
			}
			if tc.want != "" {
				if got != tc.want {
					t.Errorf("ID = %q, want %q", got, tc.want)
				}
				return
			}
			if parsed, err := uuid.Parse(got); err != nil || parsed.Version() != 7 {
				t.Errorf("ID = %q, want a new UUIDv7", got)
			}
		})
	}
}

// AccessLog writes one record per request with its route pattern, status, client IP and request ID. The
// record is written at ERROR when the status is 500 or higher.
func TestAccessLog(t *testing.T) {
	tCases := []struct {
		name      string
		target    string
		status    int
		wantRoute string
		wantLevel string
	}{
		{
			name:      "a matched request logs its route pattern",
			target:    "/v1/articles/123",
			status:    204,
			wantRoute: "GET /v1/articles/{id}",
			wantLevel: "INFO",
		},
		{
			name:      "an unmatched request logs as unmatched",
			target:    "/nope",
			status:    404,
			wantRoute: "unmatched",
			wantLevel: "INFO",
		},
		{
			name:      "a server error logs at error level",
			target:    "/v1/fail",
			status:    500,
			wantRoute: "GET /v1/fail",
			wantLevel: "ERROR",
		},
		{
			name:      "a request to a quiet path logs at debug level",
			target:    "/health",
			status:    200,
			wantRoute: "GET /health",
			wantLevel: "DEBUG",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			mux := http.NewServeMux()
			mux.HandleFunc("GET /v1/articles/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
			mux.HandleFunc("GET /v1/fail", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
			mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
			handler := httpserver.Chain(mux, httpserver.RequestID(),
				httpserver.AccessLog(newJSONLogger(t, &logs), nil, "/health", "/ready"))
			request := httptest.NewRequest("GET", tc.target, nil)
			request.RemoteAddr = "203.0.113.7:1234"
			request.Header.Set("X-Forwarded-For", "1.2.3.4") // ignored: the peer is not a trusted proxy

			handler.ServeHTTP(httptest.NewRecorder(), request)

			var line map[string]any
			if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
				t.Fatalf("log %q is not one JSON line: %v", logs.String(), err)
			}
			want := map[string]any{
				"route":     tc.wantRoute,
				"status":    float64(tc.status),
				"level":     tc.wantLevel,
				"client_ip": "203.0.113.7",
			}
			for key, value := range want {
				if line[key] != value {
					t.Errorf("%s = %v, want %v", key, line[key], value)
				}
			}
			if line["request_id"] == nil {
				t.Error("request_id is missing from the access log")
			}
		})
	}
}

// Recover turns a panic into a 500 problem that hides the panic value, which it logs instead. It lets
// http.ErrAbortHandler through.
func TestRecover(t *testing.T) {
	tCases := []struct {
		name      string
		panicWith any
		wantPanic bool // the middleware must re-panic
	}{
		{name: "a panic becomes a 500 problem with its value logged", panicWith: "boom"},
		{name: "an aborted response is re-panicked for the server", panicWith: http.ErrAbortHandler, wantPanic: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			handler := httpserver.Recover(newJSONLogger(t, &logs))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				panic(tc.panicWith)
			}))
			recorder := httptest.NewRecorder()
			if tc.wantPanic {
				defer func() {
					if recover() == nil {
						t.Error("Recover swallowed http.ErrAbortHandler, want it re-panicked")
					}
				}()
			}

			handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))

			if recorder.Code != 500 || recorder.Header().Get("Content-Type") != "application/problem+json" ||
				!strings.Contains(recorder.Body.String(), `"code":"internal"`) {
				t.Errorf("response = %d %q %s, want a 500 internal problem",
					recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body)
			}
			if strings.Contains(recorder.Body.String(), "boom") {
				t.Errorf("body = %s, want the panic value kept out", recorder.Body)
			}
			if !strings.Contains(logs.String(), "boom") {
				t.Errorf("log = %s, want the panic value", logs.String())
			}
		})
	}
}

// SecurityHeaders sets a closed CSP everywhere except under /swagger/, and HSTS only in production.
func TestSecurityHeaders(t *testing.T) {
	const apiCSP = "default-src 'none'; frame-ancestors 'none'"
	const swaggerCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'"
	tCases := []struct {
		name       string
		production bool
		path       string
		wantCSP    string
		wantHSTS   string
	}{
		{name: "an API path gets the closed CSP", path: "/v1/articles", wantCSP: apiCSP},
		{
			name:    "the Swagger UI gets a CSP that lets its scripts load",
			path:    "/swagger/index.html",
			wantCSP: swaggerCSP,
		},
		{
			name:       "production adds HSTS",
			production: true,
			path:       "/v1/articles",
			wantCSP:    apiCSP,
			wantHSTS:   "max-age=31536000; includeSubDomains",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			httpserver.SecurityHeaders(tc.production)(ok).ServeHTTP(recorder, httptest.NewRequest("GET", tc.path, nil))

			got := map[string]string{
				"Content-Security-Policy":   recorder.Header().Get("Content-Security-Policy"),
				"Strict-Transport-Security": recorder.Header().Get("Strict-Transport-Security"),
				"X-Content-Type-Options":    recorder.Header().Get("X-Content-Type-Options"),
				"X-Frame-Options":           recorder.Header().Get("X-Frame-Options"),
			}
			want := map[string]string{
				"Content-Security-Policy":   tc.wantCSP,
				"Strict-Transport-Security": tc.wantHSTS,
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("headers = %v, want %v", got, want)
			}
		})
	}
}

// CORS answers a preflight from a listed origin and gives any other origin no access. A browser may send no
// header outside the list.
func TestCORS(t *testing.T) {
	handler := httpserver.CORS([]string{"https://a.example"})(ok)
	tCases := []struct {
		name, origin string
		headers      string // Access-Control-Request-Headers, lowercase and sorted as browsers send it; "" means none
		wantAllow    string
	}{
		{name: "a listed origin is allowed", origin: "https://a.example", wantAllow: "https://a.example"},
		{name: "another origin is not", origin: "https://evil.example", wantAllow: ""},
		{name: "a header outside the list may not", origin: "https://a.example", headers: "x-internal", wantAllow: ""},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("OPTIONS", "/v1/articles", nil)
			request.Header.Set("Origin", tc.origin)
			request.Header.Set("Access-Control-Request-Method", "POST")
			if tc.headers != "" {
				request.Header.Set("Access-Control-Request-Headers", tc.headers)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != tc.wantAllow {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.wantAllow)
			}
		})
	}
}

// BodyLimit lets a body within the limit through and makes a larger one a 413 problem when decoded.
func TestBodyLimit(t *testing.T) {
	tCases := []struct {
		name       string
		body       string
		wantStatus int // 0 means the body was accepted
	}{
		{name: "a body within the limit is accepted", body: `{"a":"0"}`},
		{name: "a body over the limit is too large", body: `{"a":"0123456789"}`, wantStatus: 413},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var problem *httpserver.Problem
			handler := httpserver.BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				problem = httpserver.DecodeJSON(r, &map[string]any{})
			}))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(tc.body)))

			switch {
			case tc.wantStatus == 0 && problem != nil:
				t.Errorf("DecodeJSON = %+v, want the body accepted", *problem)
			case tc.wantStatus != 0 && (problem == nil || problem.Status != tc.wantStatus):
				t.Errorf("DecodeJSON = %+v, want status %d", problem, tc.wantStatus)
			}
		})
	}
}
