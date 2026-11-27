package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// Health answers 200 ok.
func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()

	httpserver.Health()(recorder, httptest.NewRequest("GET", "/health", nil))

	if recorder.Code != 200 || strings.TrimSpace(recorder.Body.String()) != `{"status":"ok"}` {
		t.Errorf("Health = %d %s, want 200 {\"status\":\"ok\"}", recorder.Code, recorder.Body)
	}
}

// Ready answers 200 when every check passes, and 503 naming the failed check without its error text.
func TestReady(t *testing.T) {
	probe := func(err error) func(context.Context) error { return func(context.Context) error { return err } }
	tCases := []struct {
		name       string
		checks     []httpserver.Check
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name:       "every check passing is ready",
			checks:     []httpserver.Check{{Name: "database", Probe: probe(nil)}},
			wantStatus: 200,
			wantBody:   map[string]any{"status": "ok", "checks": map[string]any{"database": "ok"}},
		},
		{
			name: "a failing check is unavailable without its error text",
			checks: []httpserver.Check{{
				Name:  "database",
				Probe: probe(errors.New("dial tcp 10.0.0.1:5432: connection refused")),
			}},
			wantStatus: 503,
			wantBody: map[string]any{
				"status": "unavailable",
				"checks": map[string]any{"database": "unavailable"},
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			httpserver.Ready(discard, time.Second, tc.checks...)(recorder, httptest.NewRequest("GET", "/ready", nil))

			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %s is not JSON: %v", recorder.Body, err)
			}
			if recorder.Code != tc.wantStatus || !reflect.DeepEqual(body, tc.wantBody) {
				t.Errorf("Ready = %d %v, want %d %v", recorder.Code, body, tc.wantStatus, tc.wantBody)
			}
		})
	}
}
