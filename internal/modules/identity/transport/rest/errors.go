package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

// problem is how the API answers one domain error: with status and code, or, when field is set, with a
// validation_failed problem that names the request field holding the refused value and the rule it breaks.
type problem struct {
	err    error
	status int
	code   string
	field  string
	rule   string
}

// problems are the answers every endpoint shares.
var problems = []problem{
	{err: domain.ErrInvalidEmail, field: "email", rule: "email"},
	{err: domain.ErrWeakPassword, field: "password", rule: "password"},
	{err: domain.ErrInvalidDisplayName, field: "display_name", rule: "display_name"},
	{err: domain.ErrNotFound, status: http.StatusNotFound, code: "not_found"},
	{err: domain.ErrEmailTaken, status: http.StatusConflict, code: "email_taken"},
	{err: domain.ErrInvalidCredentials, status: http.StatusUnauthorized, code: "invalid_credentials"},
	{err: domain.ErrEmailNotVerified, status: http.StatusForbidden, code: "email_not_verified"},
	{err: domain.ErrAccountLocked, status: http.StatusLocked, code: "account_locked"},
	{err: domain.ErrInvalidCode, status: http.StatusUnprocessableEntity, code: "invalid_code"},
	{err: domain.ErrCodeExpired, status: http.StatusUnprocessableEntity, code: "code_expired"},
	{err: domain.ErrTooManyAttempts, status: http.StatusUnprocessableEntity, code: "too_many_attempts"},
	{err: domain.ErrInvalidToken, status: http.StatusUnauthorized, code: "invalid_token"},
}

// The answers of the endpoints that differ from problems, which those endpoints pass to fail.
var (
	// weakNewPassword names the new_password field of a password change.
	weakNewPassword = problem{err: domain.ErrWeakPassword, field: "new_password", rule: "password"}
	// unusableResetToken answers a reset token the use cases refuse as unprocessable rather than
	// unauthorized: it is a value of the body, not the caller's credential as a refresh token is.
	unusableResetToken = problem{
		err:    domain.ErrInvalidToken,
		status: http.StatusUnprocessableEntity,
		code:   "invalid_token",
	}
)

// fail answers err with the first problem of the endpoint's own, then of problems, whose error it is. Any
// other error is answered 500 internal, and its cause is logged, never returned.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error, own ...problem) {
	for _, known := range slices.Concat(own, problems) {
		if errors.Is(err, known.err) {
			known.write(w, r)
			return
		}
	}

	h.log.LogAttrs(r.Context(), slog.LevelError, "identity request failed", slog.String("error", err.Error()))
	httpserver.WriteProblem(w, r, httpserver.Problem{
		Status: http.StatusInternalServerError,
		Code:   "internal",
		Detail: "internal server error",
	})
}

func (p problem) write(w http.ResponseWriter, r *http.Request) {
	if p.field != "" {
		httpserver.WriteProblem(w, r, httpserver.ValidationProblem(validate.Errors{{
			Field:   p.field,
			Rule:    p.rule,
			Message: p.err.Error(),
		}}))
		return
	}
	httpserver.WriteProblem(w, r, httpserver.Problem{Status: p.status, Code: p.code, Detail: p.err.Error()})
}

func unauthenticated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	httpserver.WriteProblem(w, r, httpserver.Problem{
		Status: http.StatusUnauthorized,
		Code:   "unauthenticated",
		Detail: "the request needs valid credentials",
	})
}
