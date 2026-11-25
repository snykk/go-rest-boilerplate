package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// registerRequest is the body of POST /v1/auth/register.
type registerRequest struct {
	Email string `json:"email" validate:"required" example:"ada@example.com"`
	// Password is 8 to 72 bytes.
	Password string `json:"password" validate:"required" example:"correct-horse-battery"`
	// DisplayName is 1 to 50 characters, none of them a control character.
	DisplayName string `json:"display_name" validate:"required" example:"Ada"`
} // @name RegisterRequest

// register creates an unverified account and mails its verification code. It answers 201 even when the message
// cannot be sent, since the user can ask for another code.
//
//	@Summary		Register an account
//	@Description	Creates an unverified account with the role reader and mails a verification code to its email. The account signs in once the email is verified with POST /auth/verification.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			account	body		registerRequest		true	"The new account"
//	@Success		201		{object}	userResponse
//	@Failure		400		{object}	httpserver.Problem	"malformed_json"
//	@Failure		409		{object}	httpserver.Problem	"email_taken"
//	@Failure		413		{object}	httpserver.Problem	"body_too_large"
//	@Failure		422		{object}	httpserver.Problem	"validation_failed"
//	@Router			/auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var body registerRequest
	if !h.decode(w, r, &body) {
		return
	}

	view, err := h.useCases.Register(r.Context(), usecase.RegisterInput{
		Email:       body.Email,
		Password:    body.Password,
		DisplayName: body.DisplayName,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toUserResponse(view))
}
