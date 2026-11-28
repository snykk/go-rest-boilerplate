package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

// Problem is an RFC 9457 problem document with the extension members code, request_id and errors.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	// Code is a stable, machine-readable error code such as "slug_taken".
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
	// Errors lists the invalid fields of a validation problem.
	Errors []validate.FieldError `json:"errors,omitempty"`
} // @name Problem

// WriteProblem writes problem as application/problem+json with status problem.Status. An empty Type becomes
// "about:blank", an empty Title the status text, and an empty RequestID the request ID of r's context.
func WriteProblem(w http.ResponseWriter, r *http.Request, problem Problem) {
	if problem.Type == "" {
		problem.Type = "about:blank"
	}
	if problem.Title == "" {
		problem.Title = http.StatusText(problem.Status)
	}
	if problem.RequestID == "" {
		problem.RequestID = requestid.From(r.Context())
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

// ValidationProblem is the 422 validation_failed problem that lists fields.
func ValidationProblem(fields validate.Errors) Problem {
	return Problem{
		Status: http.StatusUnprocessableEntity,
		Code:   "validation_failed",
		Detail: "request validation failed",
		Errors: fields,
	}
}
