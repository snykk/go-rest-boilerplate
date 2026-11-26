package httpserver_test

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

// WriteProblem writes an RFC 9457 problem carrying the request ID of the request's context.
func TestWriteProblem(t *testing.T) {
	request := httptest.NewRequest("GET", "/v1/articles/x", nil)
	request = request.WithContext(requestid.With(request.Context(), "rid-1"))
	recorder := httptest.NewRecorder()

	httpserver.WriteProblem(recorder, request, httpserver.Problem{
		Status: 404,
		Code:   "not_found",
		Detail: "article not found",
	})

	if recorder.Code != 404 {
		t.Errorf("status = %d, want 404", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", contentType)
	}
	var got map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("body %q is not JSON: %v", recorder.Body, err)
	}
	want := map[string]any{
		"type":       "about:blank",
		"title":      "Not Found",
		"status":     float64(404),
		"detail":     "article not found",
		"code":       "not_found",
		"request_id": "rid-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %v, want %v", got, want)
	}
}

// ValidationProblem is a 422 validation_failed problem carrying the field errors.
func TestValidationProblem(t *testing.T) {
	fields := validate.Errors{{Field: "title", Rule: "required", Message: "is required"}}

	got := httpserver.ValidationProblem(fields)

	if got.Status != 422 || got.Code != "validation_failed" || !reflect.DeepEqual(got.Errors, []validate.FieldError(fields)) {
		t.Errorf("ValidationProblem = %+v, want status 422, code validation_failed and the field errors", got)
	}
}
