package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// get returns one published article; a draft is not found here.
//
//	@Summary	Get a published article
//	@Tags		articles
//	@Produce	json
//	@Param		id	path		string	true	"Article ID (UUID)"
//	@Success	200	{object}	articleResponse
//	@Failure	404	{object}	httpserver.Problem	"not_found"
//	@Router		/articles/{id} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := articleID(w, r)
	if !ok {
		return
	}

	view, err := h.useCases.Get(r.Context(), id, domain.StatusPublished)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toResponse(view))
}
