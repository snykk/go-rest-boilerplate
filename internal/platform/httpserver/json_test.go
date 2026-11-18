package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// WriteJSON writes value as JSON with the given status.
func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	httpserver.WriteJSON(recorder, 201, map[string]int{"a": 1})

	if recorder.Code != 201 {
		t.Errorf("status = %d, want 201", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != `{"a":1}` {
		t.Errorf(`body = %s, want {"a":1}`, body)
	}
}

// DecodeJSON accepts exactly one JSON object with known fields. Anything else is a 400 problem, or a 413
// problem when the body is over the size limit.
func TestDecodeJSON(t *testing.T) {
	type in struct {
		A string `json:"a"`
	}
	tCases := []struct {
		name       string
		body       string
		limit      int64 // wraps the body in http.MaxBytesReader when > 0
		wantStatus int   // 0 means success
		wantCode   string
	}{
		{name: "one object with known fields is decoded", body: `{"a":"x"}`},
		{name: "an empty body is malformed", body: ``, wantStatus: 400, wantCode: "malformed_json"},
		{name: "a syntax error is malformed", body: `{"a":`, wantStatus: 400, wantCode: "malformed_json"},
		{name: "an unknown field is malformed", body: `{"a":"x","b":2}`, wantStatus: 400, wantCode: "malformed_json"},
		{
			name:       "data after the object is malformed",
			body:       `{"a":"x"} {"a":"y"}`,
			wantStatus: 400,
			wantCode:   "malformed_json",
		},
		{name: "a value of the wrong type is malformed", body: `{"a":1}`, wantStatus: 400, wantCode: "malformed_json"},
		{
			name:       "a body over the size limit is too large",
			body:       `{"a":"` + strings.Repeat("x", 100) + `"}`,
			limit:      16,
			wantStatus: 413,
			wantCode:   "body_too_large",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			if tc.limit > 0 {
				request.Body = http.MaxBytesReader(httptest.NewRecorder(), request.Body, tc.limit)
			}

			problem := httpserver.DecodeJSON(request, &in{})

			switch {
			case tc.wantStatus == 0 && problem != nil:
				t.Errorf("DecodeJSON = %+v, want nil", *problem)
			case tc.wantStatus != 0 && (problem == nil || problem.Status != tc.wantStatus || problem.Code != tc.wantCode):
				t.Errorf("DecodeJSON = %+v, want status %d, code %s", problem, tc.wantStatus, tc.wantCode)
			}
		})
	}
}
