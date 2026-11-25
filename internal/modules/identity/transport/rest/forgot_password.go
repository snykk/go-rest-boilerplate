package rest

import (
	"net/http"
)

// forgotPasswordRequest is the body of POST /v1/auth/password/forgot.
type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required" example:"ada@example.com"`
} // @name ForgotPasswordRequest

// forgotPassword mails a password reset token, unless the one sent before is younger than the cooldown. It
// answers 202 with no body whatever the email, so the answer does not tell whether an account exists or a token
// was sent.
//
//	@Summary		Ask for a password reset
//	@Description	Mails a reset token, which replaces the one sent before, when the email belongs to a verified account whose last token was issued at least IDENTITY_RESEND_COOLDOWN (60s by default) ago. The answer is the same for any email, so it does not tell whether an account exists or a token was sent.
//	@Tags			auth
//	@Accept			json
//	@Param			request	body	forgotPasswordRequest	true	"The email of the account"
//	@Success		202
//	@Failure		400	{object}	httpserver.Problem	"malformed_json"
//	@Failure		413	{object}	httpserver.Problem	"body_too_large"
//	@Failure		422	{object}	httpserver.Problem	"validation_failed"
//	@Router			/auth/password/forgot [post]
func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body forgotPasswordRequest
	if !h.decode(w, r, &body) {
		return
	}

	if err := h.useCases.ForgotPassword(r.Context(), body.Email); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
