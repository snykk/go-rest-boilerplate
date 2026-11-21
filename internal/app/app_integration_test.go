//go:build integration

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/app"
	"github.com/snykk/go-rest-boilerplate/internal/platform/lifecycle"
	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

// NewHandler serves the article lifecycle end to end on a real database, with request IDs that follow the
// caller's trace, readiness, a Swagger UI that can load its scripts, and an operator endpoint for the log level
// that exists only with an admin token and requires it.
func TestNewHandler(t *testing.T) {
	const adminToken = "0123456789abcdef0123456789abcdef"
	pool := testenv.Postgres(t, migrations.FS)
	config, err := app.LoadConfig(env(map[string]string{
		"DATABASE_URL": "postgres://unused", // the pool comes from testenv
		"ADMIN_TOKEN":  adminToken,
	}))
	if err != nil {
		t.Fatalf("LoadConfig = %v", err)
	}
	providers, err := telemetry.New(t.Context(), telemetry.Config{
		Service:     "test",
		SampleRatio: 1,
	})
	if err != nil {
		t.Fatalf("telemetry.New = %v", err)
	}
	var logs bytes.Buffer
	level := new(slog.LevelVar)
	log, err := logger.New(&logs, logger.Options{
		Level:  level,
		Format: "json",
	})
	if err != nil {
		t.Fatalf("logger.New = %v", err)
	}
	handler := app.NewHandler(config, app.Deps{
		Log:       log,
		Level:     level,
		Pool:      pool,
		Telemetry: providers,
	})
	doWith := func(handler http.Handler, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, target, strings.NewReader(body))
		for key, value := range header {
			request.Header.Set(key, value)
		}
		handler.ServeHTTP(recorder, request)
		return recorder
	}
	do := func(method, target, body string) *httptest.ResponseRecorder {
		return doWith(handler, method, target, body, nil)
	}
	field := func(recorder *httptest.ResponseRecorder, key string) any {
		var decoded map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("body %s is not JSON: %v", recorder.Body, err)
		}
		return decoded[key]
	}
	wantStatus := func(recorder *httptest.ResponseRecorder, want int, step string) {
		t.Helper()
		if recorder.Code != want {
			t.Fatalf("%s: status = %d, want %d; body %s", step, recorder.Code, want, recorder.Body)
		}
	}
	wantProblem := func(recorder *httptest.ResponseRecorder, status int, code, step string) {
		t.Helper()
		wantStatus(recorder, status, step)
		if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" ||
			field(recorder, "code") != code {
			t.Errorf("%s: %s %s, want an application/problem+json %s problem", step, contentType, recorder.Body, code)
		}
	}

	recorder := do("POST", "/v1/articles", `{"slug":"hello-world","title":"Hello","body":"Text"}`)
	wantStatus(recorder, 201, "create")
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Error("create: X-Request-ID is missing")
	}
	id := field(recorder, "id").(string)
	created := recorder.Body.String()
	if location := recorder.Header().Get("Location"); location != "/v1/articles/drafts/"+id {
		t.Errorf("create: Location = %q, want the draft's URL", location)
	}
	if got := do("GET", recorder.Header().Get("Location"), "").Body.String(); got != created {
		t.Errorf("get the draft at Location = %s, want the article as create returned it: %s", got, created)
	}
	wantProblem(do("GET", "/v1/articles/"+id, ""), 404, "not_found", "a draft on the public routes")
	if total := field(do("GET", "/v1/articles?status=draft", ""), "total"); total != float64(0) {
		t.Errorf("public list with a draft stored: total = %v, want 0", total)
	}
	if total := field(do("GET", "/v1/articles/drafts", ""), "total"); total != float64(1) {
		t.Errorf("drafts list: total = %v, want 1", total)
	}

	wantStatus(do("POST", "/v1/articles", `{"slug":"hello-world","title":"Again"}`), 409, "create with a used slug")
	wantProblem(do("POST", "/v1/articles", `{"slug":"nul","title":"t","body":"a\u0000b"}`), 422, "invalid_body",
		"create with a NUL character in the body")
	wantProblem(do("GET", "/v1/nope", ""), 404, "not_found", "a path no route matches")
	recorder = do("PUT", "/v1/articles/"+id, "")
	wantProblem(recorder, 405, "method_not_allowed", "a method the route does not take")
	if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, "PATCH") {
		t.Errorf("a method the route does not take: Allow = %q, want it to list PATCH", allow)
	}

	recorder = do("POST", "/v1/articles/"+id+"/publish", "")
	wantStatus(recorder, 200, "publish")
	if field(recorder, "status") != "published" || field(recorder, "published_at") == nil {
		t.Errorf("publish: body %s, want a published article with published_at", recorder.Body)
	}
	wantStatus(do("GET", "/v1/articles/"+id, ""), 200, "get after publish")
	wantProblem(do("GET", "/v1/articles/drafts/"+id, ""), 404, "not_found", "a published article on the drafts routes")

	wantStatus(do("PATCH", "/v1/articles/"+id, `{"slug":"renamed"}`), 409, "rename a published slug")
	if total := field(do("GET", "/v1/articles", ""), "total"); total != float64(1) {
		t.Errorf("list published: total = %v, want 1", total)
	}
	wantStatus(do("DELETE", "/v1/articles/"+id, ""), 204, "delete")
	wantStatus(do("GET", "/v1/articles/"+id, ""), 404, "get after delete")
	wantStatus(do("GET", "/ready", ""), 200, "ready")

	recorder = do("GET", "/swagger/index.html", "")
	wantStatus(recorder, 200, "swagger UI")
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
		t.Errorf("swagger UI CSP = %q, want scripts allowed from 'self'", csp)
	}
	wantStatus(do("GET", "/swagger/doc.json", ""), 200, "swagger spec")

	const callerTrace = "0af7651916cd43dd8448eb211c80319c"
	logs.Reset()
	recorder = doWith(handler, "GET", "/v1/articles", "",
		map[string]string{"traceparent": "00-" + callerTrace + "-b7ad6b7169203331-01"})
	wantStatus(recorder, 200, "list in the caller's trace")
	if requestID := recorder.Header().Get("X-Request-ID"); requestID != callerTrace {
		t.Errorf("list in the caller's trace: X-Request-ID = %q, want the trace ID %q", requestID, callerTrace)
	}
	if !strings.Contains(logs.String(), `"trace_id":"`+callerTrace+`"`) {
		t.Errorf("list in the caller's trace: logs %s, want the trace ID in the access log", logs.String())
	}

	wantProblem(do("PUT", "/admin/log-level", `{"level":"debug"}`), 401, "unauthenticated", "log level without the token")
	recorder = doWith(handler, "PUT", "/admin/log-level", `{"level":"debug"}`,
		map[string]string{"Authorization": "Bearer " + adminToken})
	wantStatus(recorder, 200, "log level with the token")
	if level.Level() != slog.LevelDebug {
		t.Errorf("log level with the token: level = %s, want DEBUG", level.Level())
	}

	config.Platform.Admin.Token = ""
	withoutAdmin := app.NewHandler(config, app.Deps{
		Log:       log,
		Level:     level,
		Pool:      pool,
		Telemetry: providers,
	})
	wantStatus(doWith(withoutAdmin, "GET", "/admin/log-level", "", nil), 404, "log level without an admin token configured")
}

// App.Run gives the components the HTTP shutdown timeout and a moment more to stop, so a component that stops on
// that deadline, as the HTTP server does when requests outlive it, reports its own error rather than a timeout.
// A component that does not stop is reported with lifecycle.ErrStopTimeout. Run closes the resources after the
// components.
func TestAppRun(t *testing.T) {
	const shutdown = 300 * time.Millisecond
	errLateStop := errors.New("stopped on the shutdown deadline")
	tCases := []struct {
		name    string
		stop    time.Duration // how long the component takes to return once its context ends
		never   bool          // the component does not return
		result  error         // what the component returns
		wantErr error         // nil means Run returns nil
		notErr  error         // an error Run must not report
	}{
		{name: "a component that stops in time gives nil"},
		{
			name:    "a component that stops on the shutdown deadline reports its own error, not a timeout",
			stop:    shutdown + 50*time.Millisecond,
			result:  errLateStop,
			wantErr: errLateStop,
			notErr:  lifecycle.ErrStopTimeout,
		},
		{
			name:    "a component that does not stop is reported as late",
			never:   true,
			wantErr: lifecycle.ErrStopTimeout,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := app.LoadConfig(env(map[string]string{
				"DATABASE_URL":          testenv.Postgres(t, nil).Config().ConnString(),
				"HTTP_SHUTDOWN_TIMEOUT": shutdown.String(),
			}))
			if err != nil {
				t.Fatalf("LoadConfig = %v", err)
			}
			application, err := app.New(t.Context(), config)
			if err != nil {
				t.Fatalf("New = %v", err)
			}
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			component := lifecycle.ComponentFunc(func(ctx context.Context) error {
				<-ctx.Done()
				if tc.never {
					<-release
				}
				time.Sleep(tc.stop)
				return tc.result
			})
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			err = application.Run(ctx, component)

			if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
				t.Errorf("Run = %v, want %v", err, tc.wantErr)
			}
			if tc.notErr != nil && errors.Is(err, tc.notErr) {
				t.Errorf("Run = %v, want no %v", err, tc.notErr)
			}
		})
	}
}
