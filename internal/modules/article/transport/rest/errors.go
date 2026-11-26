package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

var domainProblems = []struct {
	err    error
	status int
	code   string
}{
	{domain.ErrNotFound, http.StatusNotFound, "not_found"},
	{domain.ErrSlugTaken, http.StatusConflict, "slug_taken"},
	{domain.ErrSlugLocked, http.StatusConflict, "slug_locked"},
	{domain.ErrAlreadyPublished, http.StatusConflict, "already_published"},
	{domain.ErrInvalidSlug, http.StatusUnprocessableEntity, "invalid_slug"},
	{domain.ErrInvalidTitle, http.StatusUnprocessableEntity, "invalid_title"},
	{domain.ErrInvalidBody, http.StatusUnprocessableEntity, "invalid_body"},
	{domain.ErrBodyTooLong, http.StatusUnprocessableEntity, "body_too_long"},
	{domain.ErrEmptyBody, http.StatusUnprocessableEntity, "empty_body"},
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
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

	h.log.LogAttrs(r.Context(), slog.LevelError, "article request failed", slog.String("error", err.Error()))
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
