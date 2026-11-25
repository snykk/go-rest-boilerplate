package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// me answers with the signed-in caller's profile.
//
//	@Summary	Get the caller's profile
//	@Tags		users
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	userResponse
//	@Failure	401	{object}	httpserver.Problem	"unauthenticated"
//	@Failure	404	{object}	httpserver.Problem	"not_found"
//	@Router		/users/me [get]
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}

	view, err := h.useCases.Me(r.Context(), userID)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toUserResponse(view))
}
