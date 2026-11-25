package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

// Handler serves the identity endpoints.
type Handler struct {
	useCases UseCases
	log      *slog.Logger
}

// NewHandler returns a Handler over useCases that logs unexpected errors to log.
func NewHandler(useCases UseCases, log *slog.Logger) *Handler {
	return &Handler{useCases: useCases, log: log}
}

// decode reads the request body into body, a pointer to a request type, and checks its validate tags. When the
// body is malformed or invalid, it answers with the problem and returns false.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, body any) bool {
	if problem := httpserver.DecodeJSON(r, body); problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return false
	}

	err := validate.Struct(body)
	if err == nil {
		return true
	}
	if fields, ok := errors.AsType[validate.Errors](err); ok {
		httpserver.WriteProblem(w, r, httpserver.ValidationProblem(fields))
		return false
	}
	h.fail(w, r, err)
	return false
}

// caller returns the ID of the signed-in user a request comes from. It answers 401 unauthenticated, and returns
// false, for a request with no caller, as on a route mounted without authentication.
func caller(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		unauthenticated(w, r)
		return "", false
	}
	return principal.ID, true
}
