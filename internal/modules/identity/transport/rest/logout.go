package rest

import (
	"net/http"
)

// logoutRequest is the body of POST /v1/auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"Zx8kQm4pR2v9sT1uW3yA5bC7dE0fG6hJ8kL2mN4pQ6r"`
} // @name LogoutRequest

// logout ends the sign-in the refresh token belongs to. It answers 204 also for a token no session has, so
// signing out twice is harmless.
//
//	@Summary		Sign out
//	@Description	Revokes the refresh token and every token rotated from the same sign-in. An access token stays valid until it expires.
//	@Tags			auth
//	@Accept			json
//	@Param			request	body	logoutRequest	true	"The refresh token"
//	@Success		204
//	@Failure		400	{object}	httpserver.Problem	"malformed_json"
//	@Failure		413	{object}	httpserver.Problem	"body_too_large"
//	@Failure		422	{object}	httpserver.Problem	"validation_failed"
//	@Router			/auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var body logoutRequest
	if !h.decode(w, r, &body) {
		return
	}

	if err := h.useCases.Logout(r.Context(), body.RefreshToken); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
