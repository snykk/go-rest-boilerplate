package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// getDraft returns one draft, for whoever may change articles; a published article is not found here.
//
//	@Summary	Get a draft
//	@Tags		articles
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Article ID (UUID)"
//	@Success	200	{object}	articleResponse
//	@Failure	401	{object}	httpserver.Problem	"unauthenticated"
//	@Failure	403	{object}	httpserver.Problem	"forbidden"
//	@Failure	404	{object}	httpserver.Problem	"not_found"
//	@Router		/articles/drafts/{id} [get]
func (h *Handler) getDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := articleID(w, r)
	if !ok {
		return
	}

	view, err := h.useCases.Get(r.Context(), id, domain.StatusDraft)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toResponse(view))
}
