package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// listDrafts returns one page of drafts, newest first, for whoever may change articles.
//
//	@Summary	List drafts
//	@Tags		articles
//	@Produce	json
//	@Param		limit	query		int		false	"Page size; 0 or none means the server's default (20 unless configured), and a value above the server's maximum (100 unless configured) means the maximum"	minimum(0)
//	@Param		offset	query		int		false	"Drafts to skip"	minimum(0)
//	@Success	200		{object}	listResponse
//	@Failure	400		{object}	httpserver.Problem	"invalid_query"
//	@Router		/articles/drafts [get]
func (h *Handler) listDrafts(w http.ResponseWriter, r *http.Request) {
	limit, offset, problem := pageInput(r.URL.Query())
	if problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return
	}

	output, err := h.useCases.List(r.Context(), usecase.ListInput{
		Status: domain.StatusDraft,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toListResponse(output))
}
