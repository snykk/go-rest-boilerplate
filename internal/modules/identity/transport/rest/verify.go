package rest

import (
	"net/http"
)

// verifyRequest is the body of POST /v1/auth/verification.
type verifyRequest struct {
	Email string `json:"email" validate:"required" example:"ada@example.com"`
	// Code is the six digits mailed to the email.
	Code string `json:"code" validate:"required" example:"042917"`
} // @name VerifyRequest

// verify marks the email verified when the code is the one mailed to it. A wrong code counts against the code,
// which ends once it has been guessed at too often or has expired.
//
//	@Summary		Verify an email
//	@Description	Marks the email verified when the code is the one mailed to it. A wrong code counts as an attempt; an expired code, or one guessed at too often, is spent, and the user asks for a new one.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	verifyRequest	true	"The email and its code"
//	@Success		204
//	@Failure		400	{object}	httpserver.Problem	"malformed_json"
//	@Failure		413	{object}	httpserver.Problem	"body_too_large"
//	@Failure		422	{object}	httpserver.Problem	"validation_failed, invalid_code, code_expired or too_many_attempts"
//	@Router			/auth/verification [post]
func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	var body verifyRequest
	if !h.decode(w, r, &body) {
		return
	}

	if err := h.useCases.Verify(r.Context(), body.Email, body.Code); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
