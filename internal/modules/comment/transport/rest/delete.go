package rest

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	identitycontract "github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
)

// delete removes a comment for its author or for an editor.
//
//	@Summary		Delete a comment
//	@Description	The comment's author may delete it, and so may an editor; anyone else is forbidden.
//	@Tags			comments
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Comment ID (UUID)"
//	@Success		204
//	@Failure		401	{object}	httpserver.Problem	"unauthenticated"
//	@Failure		403	{object}	httpserver.Problem	"forbidden"
//	@Failure		404	{object}	httpserver.Problem	"not_found"
//	@Router			/comments/{id} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	principal, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		notFound(w, r)
		return
	}

	err = h.useCases.Delete(r.Context(), usecase.DeleteInput{
		ID:             id.String(),
		CallerID:       principal.ID,
		CallerIsEditor: principal.HasRole(identitycontract.RoleEditor),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
