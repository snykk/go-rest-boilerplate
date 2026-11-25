package rest

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// list returns one page of the comments on a published article, newest first, of one author when the query
// names one.
//
//	@Summary		List the comments on a published article
//	@Description	An article that does not exist or is a draft is not_found, never an empty page.
//	@Tags			comments
//	@Produce		json
//	@Param			article_id	query		string	true	"ID (UUID) of a published article"
//	@Param			author_id	query		string	false	"Only the comments of the user with this ID (UUID)"
//	@Param			limit		query		int		false	"Page size; 0 or none means the server's default (20 unless configured), and a value above the server's maximum (100 unless configured) means the maximum"	minimum(0)
//	@Param			offset		query		int		false	"Comments to skip"	minimum(0)
//	@Success		200			{object}	listResponse
//	@Failure		400			{object}	httpserver.Problem	"invalid_query"
//	@Failure		404			{object}	httpserver.Problem	"not_found"
//	@Router			/comments [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	input, problem := listInput(r.URL.Query())
	if problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return
	}

	output, err := h.useCases.List(r.Context(), input)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toListResponse(output))
}

// listInput reads the list's query: article_id, a UUID; author_id, a UUID or none; and limit and offset, each a
// whole number of 0 or more or none. It returns an invalid_query problem, which never repeats the value, for
// any other, and the IDs in their canonical form.
func listInput(query url.Values) (usecase.ListInput, *httpserver.Problem) {
	articleID, err := uuid.Parse(query.Get("article_id"))
	if err != nil {
		return usecase.ListInput{}, invalidQuery("article_id must be the UUID of an article")
	}
	input := usecase.ListInput{ArticleID: articleID.String()}
	if value := query.Get("author_id"); value != "" {
		authorID, err := uuid.Parse(value)
		if err != nil {
			return usecase.ListInput{}, invalidQuery("author_id must be the UUID of a user")
		}
		input.AuthorID = authorID.String()
	}

	var ok bool
	if input.Limit, ok = nonNegative(query.Get("limit")); !ok {
		return usecase.ListInput{}, invalidQuery("limit must be a whole number of 0 or more")
	}
	if input.Offset, ok = nonNegative(query.Get("offset")); !ok {
		return usecase.ListInput{}, invalidQuery("offset must be a whole number of 0 or more")
	}
	return input, nil
}

func nonNegative(value string) (int, bool) {
	if value == "" {
		return 0, true
	}
	number, err := strconv.Atoi(value)
	return number, err == nil && number >= 0
}
