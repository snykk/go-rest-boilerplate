package httpserver_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// LogLevel answers the current level on GET, and on PUT sets a level slog knows and answers it, logging the
// change; it refuses a level slog does not know without changing anything.
func TestLogLevel(t *testing.T) {
	tCases := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantAnswer string // the level the response names; "" means none
		wantCode   string // the problem code; "" means no problem
		wantLevel  slog.Level
		wantLogged bool // the change is logged
	}{
		{
			name:       "GET answers the current level",
			method:     http.MethodGet,
			wantStatus: 200,
			wantAnswer: "INFO",
			wantLevel:  slog.LevelInfo,
		},
		{
			name:       "PUT sets a known level, in any case, and logs the change",
			method:     http.MethodPut,
			body:       `{"level":"debug"}`,
			wantStatus: 200,
			wantAnswer: "DEBUG",
			wantLevel:  slog.LevelDebug,
			wantLogged: true,
		},
		{
			name:       "PUT with an unknown level is refused",
			method:     http.MethodPut,
			body:       `{"level":"loud"}`,
			wantStatus: 422,
			wantCode:   "invalid_level",
			wantLevel:  slog.LevelInfo,
		},
		{
			name:       "PUT with a body that is not JSON is refused",
			method:     http.MethodPut,
			body:       `level=debug`,
			wantStatus: 400,
			wantCode:   "malformed_json",
			wantLevel:  slog.LevelInfo,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			level := new(slog.LevelVar)
			handler := httpserver.LogLevel(level, newJSONLogger(t, &logs))
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, "/admin/log-level", strings.NewReader(tc.body)))

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			var body struct{ Level, Code string }
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %s is not JSON: %v", recorder.Body, err)
			}
			if body.Level != tc.wantAnswer || body.Code != tc.wantCode {
				t.Errorf("body = level %q, code %q; want %q, %q", body.Level, body.Code, tc.wantAnswer, tc.wantCode)
			}
			if level.Level() != tc.wantLevel {
				t.Errorf("level = %s, want %s", level.Level(), tc.wantLevel)
			}
			if logged := strings.Contains(logs.String(), "log level changed"); logged != tc.wantLogged {
				t.Errorf("logged = %t, want %t; logs %s", logged, tc.wantLogged, logs.String())
			}
		})
	}
}
