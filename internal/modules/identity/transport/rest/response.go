package rest

import (
	"net/http"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// userResponse is a user as the API returns them. It never carries a password or its hash.
type userResponse struct {
	ID          string `json:"id" example:"0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"`
	Email       string `json:"email" example:"ada@example.com"`
	DisplayName string `json:"display_name" example:"Ada"`
	// Roles is an empty array, never null, for a user with no role.
	Roles []string `json:"roles" example:"reader"`
	// Verified is true once the user has confirmed their email with the code they were sent.
	Verified  bool      `json:"verified" example:"false"`
	CreatedAt time.Time `json:"created_at"`
} // @name User

func toUserResponse(view usecase.UserView) userResponse {
	roles := view.Roles
	if roles == nil {
		roles = []string{}
	}
	return userResponse{
		ID:          view.ID,
		Email:       view.Email,
		DisplayName: view.DisplayName,
		Roles:       roles,
		Verified:    view.Verified,
		CreatedAt:   view.CreatedAt,
	}
}

// tokenResponse is a token pair as the API returns it.
type tokenResponse struct {
	// AccessToken goes with each request as "Authorization: Bearer <access_token>" until it expires.
	AccessToken     string    `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	AccessExpiresAt time.Time `json:"access_expires_at"`
	// RefreshToken gets the next pair from POST /auth/refresh, once: the refresh spends it.
	RefreshToken     string    `json:"refresh_token" example:"Zx8kQm4pR2v9sT1uW3yA5bC7dE0fG6hJ8kL2mN4pQ6r"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	TokenType        string    `json:"token_type" enums:"Bearer" example:"Bearer"`
} // @name TokenPair

// writeTokens answers 200 with pair, which no cache may keep.
func writeTokens(w http.ResponseWriter, pair usecase.TokenPair) {
	w.Header().Set("Cache-Control", "no-store")
	httpserver.WriteJSON(w, http.StatusOK, tokenResponse{
		AccessToken:      pair.AccessToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshToken:     pair.RefreshToken,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		TokenType:        "Bearer",
	})
}
