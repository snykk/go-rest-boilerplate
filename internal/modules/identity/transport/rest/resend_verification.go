package rest

import (
	"net/http"
)

// resendVerificationRequest is the body of POST /v1/auth/verification/resend.
type resendVerificationRequest struct {
	Email string `json:"email" validate:"required" example:"ada@example.com"`
} // @name ResendVerificationRequest

// resendVerification mails a new verification code, which replaces the one sent before, unless that one is
// younger than the cooldown. It answers 202 with no body whatever the email, so the answer does not tell whether
// an account exists or a code was sent.
//
//	@Summary		Ask for a new verification code
//	@Description	Mails a new code, which replaces the one sent before, when the email belongs to an unverified account whose last code was issued at least IDENTITY_RESEND_COOLDOWN (60s by default) ago. The answer is the same for any email, so it does not tell whether an account exists or a code was sent.
//	@Tags			auth
//	@Accept			json
//	@Param			request	body	resendVerificationRequest	true	"The email to verify"
//	@Success		202
//	@Failure		400	{object}	httpserver.Problem	"malformed_json"
//	@Failure		413	{object}	httpserver.Problem	"body_too_large"
//	@Failure		422	{object}	httpserver.Problem	"validation_failed"
//	@Router			/auth/verification/resend [post]
func (h *Handler) resendVerification(w http.ResponseWriter, r *http.Request) {
	var body resendVerificationRequest
	if !h.decode(w, r, &body) {
		return
	}

	if err := h.useCases.ResendVerification(r.Context(), body.Email); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
