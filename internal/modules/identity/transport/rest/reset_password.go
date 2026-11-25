package rest

import (
	"net/http"
)

// resetPasswordRequest is the body of POST /v1/auth/password/reset.
type resetPasswordRequest struct {
	// Token is the reset token mailed to the account.
	Token string `json:"token" validate:"required" example:"Zx8kQm4pR2v9sT1uW3yA5bC7dE0fG6hJ8kL2mN4pQ6r"`
	// Password is the new password, 8 to 72 bytes.
	Password string `json:"password" validate:"required" example:"battery-staple-horse"`
} // @name ResetPasswordRequest

// resetPassword sets a new password with a reset token, which it spends, ends any lockout of the account and
// signs it out everywhere. A token it refuses is a value of the body, so it answers 422 invalid_token rather
// than 401.
//
//	@Summary		Reset the password
//	@Description	Sets the new password with the mailed token, spends the token, ends any lockout of the account and revokes every refresh token of the account.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	resetPasswordRequest	true	"The token and the new password"
//	@Success		204
//	@Failure		400	{object}	httpserver.Problem	"malformed_json"
//	@Failure		413	{object}	httpserver.Problem	"body_too_large"
//	@Failure		422	{object}	httpserver.Problem	"validation_failed or invalid_token"
//	@Router			/auth/password/reset [post]
func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body resetPasswordRequest
	if !h.decode(w, r, &body) {
		return
	}

	if err := h.useCases.ResetPassword(r.Context(), body.Token, body.Password); err != nil {
		h.fail(w, r, err, unusableResetToken)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
