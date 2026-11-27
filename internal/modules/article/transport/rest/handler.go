package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

// Handler serves the article endpoints.
type Handler struct {
	useCases UseCases
	log      *slog.Logger
}

// NewHandler returns a Handler over useCases that logs unexpected errors to log.
func NewHandler(useCases UseCases, log *slog.Logger) *Handler {
	return &Handler{useCases: useCases, log: log}
}

func (h *Handler) valid(w http.ResponseWriter, r *http.Request, body any) bool {
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

func articleID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		notFound(w, r)
		return "", false
	}
	return id.String(), true
}

func pageInput(query url.Values) (limit, offset int, problem *httpserver.Problem) {
	var ok bool
	if limit, ok = nonNegative(query.Get("limit")); !ok {
		return 0, 0, invalidQuery("limit must be a whole number of 0 or more")
	}
	if offset, ok = nonNegative(query.Get("offset")); !ok {
		return 0, 0, invalidQuery("offset must be a whole number of 0 or more")
	}
	return limit, offset, nil
}

func nonNegative(value string) (int, bool) {
	if value == "" {
		return 0, true
	}
	number, err := strconv.Atoi(value)
	return number, err == nil && number >= 0
}

func invalidQuery(detail string) *httpserver.Problem {
	return &httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_query", Detail: detail}
}
