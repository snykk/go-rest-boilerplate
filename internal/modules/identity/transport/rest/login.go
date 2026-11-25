package rest

import (
	"net/http"
)

// loginRequest is the body of POST /v1/auth/login.
type loginRequest struct {
	Email    string `json:"email" validate:"required" example:"ada@example.com"`
	Password string `json:"password" validate:"required" example:"correct-horse-battery"`
} // @name LoginRequest

// login signs a verified user in and answers with their first token pair.
//
//	@Summary		Sign in
//	@Description	Returns an access token and a refresh token for a verified account. Wrong passwords lock the account for a while once too many have been tried.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		loginRequest		true	"The email and password"
//	@Success		200			{object}	tokenResponse
//	@Failure		400			{object}	httpserver.Problem	"malformed_json"
//	@Failure		401			{object}	httpserver.Problem	"invalid_credentials"
//	@Failure		403			{object}	httpserver.Problem	"email_not_verified"
//	@Failure		413			{object}	httpserver.Problem	"body_too_large"
//	@Failure		422			{object}	httpserver.Problem	"validation_failed"
//	@Failure		423			{object}	httpserver.Problem	"account_locked"
//	@Router			/auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if !h.decode(w, r, &body) {
		return
	}

	pair, err := h.useCases.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	writeTokens(w, pair)
}
