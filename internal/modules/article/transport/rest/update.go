package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// updateRequest is the body of PATCH /v1/articles/{id}. A field that is absent or null stays as it is.
type updateRequest struct {
	Slug  *string `json:"slug" example:"hello-again"`
	Title *string `json:"title" example:"Hello again"`
	Body  *string `json:"body" example:"An edited article."`
} // @name UpdateArticleRequest

// update changes the fields present in the body.
//
//	@Summary	Update an article
//	@Tags		articles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string			true	"Article ID (UUID)"
//	@Param		changes	body		updateRequest	true	"The fields to change"
//	@Success	200		{object}	articleResponse
//	@Failure	400		{object}	httpserver.Problem	"malformed_json"
//	@Failure	401		{object}	httpserver.Problem	"unauthenticated"
//	@Failure	403		{object}	httpserver.Problem	"forbidden"
//	@Failure	404		{object}	httpserver.Problem	"not_found"
//	@Failure	409		{object}	httpserver.Problem	"slug_taken or slug_locked"
//	@Failure	413		{object}	httpserver.Problem	"body_too_large"
//	@Failure	422		{object}	httpserver.Problem	"invalid_slug, invalid_title, invalid_body or body_too_long"
//	@Router		/articles/{id} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := articleID(w, r)
	if !ok {
		return
	}
	var body updateRequest
	if problem := httpserver.DecodeJSON(r, &body); problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return
	}

	view, err := h.useCases.Update(r.Context(), usecase.UpdateInput{
		ID:    id,
		Slug:  body.Slug,
		Title: body.Title,
		Body:  body.Body,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toResponse(view))
}
