package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// list returns one page of published articles, newest first. Drafts are never listed here.
//
//	@Summary	List published articles
//	@Tags		articles
//	@Produce	json
//	@Param		limit	query		int		false	"Page size; 0 or none means the server's default (20 unless configured), and a value above the server's maximum (100 unless configured) means the maximum"	minimum(0)
//	@Param		offset	query		int		false	"Articles to skip"	minimum(0)
//	@Success	200		{object}	listResponse
//	@Failure	400		{object}	httpserver.Problem	"invalid_query"
//	@Router		/articles [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, problem := pageInput(r.URL.Query())
	if problem != nil {
		httpserver.WriteProblem(w, r, *problem)
		return
	}

	output, err := h.useCases.List(r.Context(), usecase.ListInput{
		Status: domain.StatusPublished,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toListResponse(output))
}
