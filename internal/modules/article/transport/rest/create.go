package rest

import (
	"net/http"
	"path"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// createRequest is the body of POST /v1/articles.
type createRequest struct {
	Slug  string `json:"slug" validate:"required" example:"hello-world"`
	Title string `json:"title" validate:"required" example:"Hello, world"`
	Body  string `json:"body" example:"The first article."`
} // @name CreateArticleRequest

// create stores a new draft article and answers with the location of the draft, {prefix}/articles/drafts/{id},
// under the prefix the routes are mounted on.
//
//	@Summary	Create a draft article
//	@Tags		articles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		article	body		createRequest		true	"The new article"
//	@Success	201		{object}	articleResponse
//	@Header		201		{string}	Location	"/v1/articles/drafts/{id}"
//	@Failure	400		{object}	httpserver.Problem	"malformed_json"
//	@Failure	401		{object}	httpserver.Problem	"unauthenticated"
//	@Failure	403		{object}	httpserver.Problem	"forbidden"
//	@Failure	409		{object}	httpserver.Problem	"slug_taken"
//	@Failure	413		{object}	httpserver.Problem	"body_too_large"
//	@Failure	422		{object}	httpserver.Problem	"validation_failed, invalid_slug, invalid_title, invalid_body or body_too_long"
//	@Router		/articles [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body createRequest
	if problem := httpserver.DecodeJSON(r, &body); problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return
	}
	if !h.valid(w, r, body) {
		return
	}

	view, err := h.useCases.Create(r.Context(), usecase.CreateInput{
		Slug:  body.Slug,
		Title: body.Title,
		Body:  body.Body,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.Header().Set("Location", path.Join(r.URL.Path, "drafts", view.ID))
	httpserver.WriteJSON(w, http.StatusCreated, toResponse(view))
}
