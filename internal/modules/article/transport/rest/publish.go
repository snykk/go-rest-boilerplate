package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// publish publishes a draft.
//
//	@Summary	Publish an article
//	@Tags		articles
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Article ID (UUID)"
//	@Success	200	{object}	articleResponse
//	@Failure	401	{object}	httpserver.Problem	"unauthenticated"
//	@Failure	403	{object}	httpserver.Problem	"forbidden"
//	@Failure	404	{object}	httpserver.Problem	"not_found"
//	@Failure	409	{object}	httpserver.Problem	"already_published"
//	@Failure	422	{object}	httpserver.Problem	"empty_body"
//	@Router		/articles/{id}/publish [post]
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	id, ok := articleID(w, r)
	if !ok {
		return
	}

	view, err := h.useCases.Publish(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toResponse(view))
}
