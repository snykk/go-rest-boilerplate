package rest

import (
	"net/http"
	"strings"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

// addRequest is the body of POST /v1/comments.
type addRequest struct {
	// ArticleID is the ID of a published article.
	ArticleID string `json:"article_id" validate:"required,uuid" example:"0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c45"`
	// Body is 1 to 2000 characters once trimmed, none of them NUL.
	Body string `json:"body" validate:"required" example:"Thanks, this helped."`
} // @name AddCommentRequest

// add stores a comment by the signed-in caller on a published article.
//
//	@Summary		Comment on a published article
//	@Description	The caller is the author, and the comment keeps the display name the author has now. An article that does not exist or is a draft is not_found, and so is a caller whose account is gone.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			comment	body		addRequest	true	"The article and the text"
//	@Success		201		{object}	commentResponse
//	@Failure		400		{object}	httpserver.Problem	"malformed_json"
//	@Failure		401		{object}	httpserver.Problem	"unauthenticated"
//	@Failure		404		{object}	httpserver.Problem	"not_found"
//	@Failure		413		{object}	httpserver.Problem	"body_too_large"
//	@Failure		422		{object}	httpserver.Problem	"validation_failed"
//	@Router			/comments [post]
func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	principal, ok := caller(w, r)
	if !ok {
		return
	}
	var body addRequest
	if problem := httpserver.DecodeJSON(r, &body); problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return
	}
	if err := validate.Struct(body); err != nil {
		h.fail(w, r, err)
		return
	}

	view, err := h.useCases.Add(r.Context(), usecase.AddInput{
		ArticleID: strings.ToLower(body.ArticleID),
		AuthorID:  principal.ID,
		Body:      body.Body,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toResponse(view))
}
