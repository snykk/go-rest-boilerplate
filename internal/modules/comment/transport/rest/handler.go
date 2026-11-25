package rest

import (
	"log/slog"
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
)

// Handler serves the comment endpoints.
type Handler struct {
	useCases UseCases
	log      *slog.Logger
}

// NewHandler returns a Handler over useCases that logs unexpected errors to log.
func NewHandler(useCases UseCases, log *slog.Logger) *Handler {
	return &Handler{useCases: useCases, log: log}
}

// caller returns the signed-in user a request comes from. It answers 401 unauthenticated, and returns false,
// for a request with no caller, as on a route mounted without authentication.
func caller(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		unauthenticated(w, r)
		return auth.Principal{}, false
	}
	return principal, true
}
