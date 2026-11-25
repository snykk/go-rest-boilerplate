package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

var domainProblems = []struct {
	err    error
	status int
	code   string
}{
	{domain.ErrNotFound, http.StatusNotFound, "not_found"},
	{domain.ErrArticleNotFound, http.StatusNotFound, "not_found"},
	{domain.ErrAuthorNotFound, http.StatusNotFound, "not_found"},
	{domain.ErrForbidden, http.StatusForbidden, "forbidden"},
}

// fail answers err: a request that fails validation, or a body the comment rules refuse, with a 422
// validation_failed problem; a domain error with its problem; and any other error with 500 internal, its cause
// logged and never returned.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	if fields, ok := errors.AsType[validate.Errors](err); ok {
		httpserver.WriteProblem(w, r, httpserver.ValidationProblem(fields))
		return
	}
	if errors.Is(err, domain.ErrInvalidBody) {
		httpserver.WriteProblem(w, r, httpserver.ValidationProblem(validate.Errors{{
			Field:   "body",
			Rule:    "body",
			Message: domain.ErrInvalidBody.Error(),
		}}))
		return
	}
	for _, problem := range domainProblems {
		if errors.Is(err, problem.err) {
			httpserver.WriteProblem(w, r, httpserver.Problem{
				Status: problem.status,
				Code:   problem.code,
				Detail: problem.err.Error(),
			})
			return
		}
	}

	h.log.LogAttrs(r.Context(), slog.LevelError, "comment request failed", slog.String("error", err.Error()))
	httpserver.WriteProblem(w, r, httpserver.Problem{
		Status: http.StatusInternalServerError,
		Code:   "internal",
		Detail: "internal server error",
	})
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteProblem(w, r, httpserver.Problem{
		Status: http.StatusNotFound,
		Code:   "not_found",
		Detail: domain.ErrNotFound.Error(),
	})
}

func unauthenticated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	httpserver.WriteProblem(w, r, httpserver.Problem{
		Status: http.StatusUnauthorized,
		Code:   "unauthenticated",
		Detail: "the request needs valid credentials",
	})
}

func invalidQuery(detail string) *httpserver.Problem {
	return &httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_query", Detail: detail}
}
