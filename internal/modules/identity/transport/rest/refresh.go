package rest

import (
	"net/http"
)

// refreshRequest is the body of POST /v1/auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"Zx8kQm4pR2v9sT1uW3yA5bC7dE0fG6hJ8kL2mN4pQ6r"`
} // @name RefreshRequest

// refresh spends a refresh token and answers with the next token pair.
//
//	@Summary		Refresh the tokens
//	@Description	Spends the refresh token and returns the next pair. A token spent moments ago, as by another tab refreshing at the same time, is refused and changes nothing; a token spent earlier is taken as stolen, and every token of its sign-in is revoked.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		refreshRequest		true	"The refresh token"
//	@Success		200		{object}	tokenResponse
//	@Failure		400		{object}	httpserver.Problem	"malformed_json"
//	@Failure		401		{object}	httpserver.Problem	"invalid_token"
//	@Failure		413		{object}	httpserver.Problem	"body_too_large"
//	@Failure		422		{object}	httpserver.Problem	"validation_failed"
//	@Router			/auth/refresh [post]
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var body refreshRequest
	if !h.decode(w, r, &body) {
		return
	}

	pair, err := h.useCases.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	writeTokens(w, pair)
}
