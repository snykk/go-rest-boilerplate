package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// changePasswordRequest is the body of POST /v1/auth/password/change.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required" example:"correct-horse-battery"`
	// NewPassword is 8 to 72 bytes.
	NewPassword string `json:"new_password" validate:"required" example:"battery-staple-horse"`
} // @name ChangePasswordRequest

// changePassword sets a new password for the signed-in caller, who gives the current one, and answers with the
// token pair of a new sign-in.
//
//	@Summary		Change the password
//	@Description	Sets the new password when the current one is right, revokes every refresh token of the account and returns the pair of a new sign-in. A wrong current password counts toward the lockout as a failed sign-in does.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		changePasswordRequest	true	"The current and the new password"
//	@Success		200		{object}	tokenResponse
//	@Failure		400		{object}	httpserver.Problem	"malformed_json"
//	@Failure		401		{object}	httpserver.Problem	"unauthenticated or invalid_credentials"
//	@Failure		404		{object}	httpserver.Problem	"not_found"
//	@Failure		413		{object}	httpserver.Problem	"body_too_large"
//	@Failure		422		{object}	httpserver.Problem	"validation_failed"
//	@Failure		423		{object}	httpserver.Problem	"account_locked"
//	@Router			/auth/password/change [post]
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var body changePasswordRequest
	if !h.decode(w, r, &body) {
		return
	}

	pair, err := h.useCases.ChangePassword(r.Context(), usecase.ChangePasswordInput{
		UserID:  userID,
		Current: body.CurrentPassword,
		New:     body.NewPassword,
	})
	if err != nil {
		h.fail(w, r, err, weakNewPassword)
		return
	}

	writeTokens(w, pair)
}
