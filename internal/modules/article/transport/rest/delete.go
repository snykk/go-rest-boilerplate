package rest

import (
	"net/http"
)

// delete removes an article.
//
//	@Summary	Delete an article
//	@Tags		articles
//	@Param		id	path	string	true	"Article ID (UUID)"
//	@Success	204
//	@Failure	404	{object}	httpserver.Problem	"not_found"
//	@Router		/articles/{id} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := articleID(w, r)
	if !ok {
		return
	}

	if err := h.useCases.Delete(r.Context(), id); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
