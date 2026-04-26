// Package users serves the /users/* HTTP endpoints — anything scoped
// to the authenticated user's own profile / data. Auth flows live in
// the sibling package internal/http/handlers/v1/auth.
package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	httpauth "github.com/snykk/go-rest-boilerplate/internal/http/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
)

// Handler serves user-domain endpoints. It calls into users.Usecase
// only — never directly into the repository or auth context.
type Handler struct {
	usecase users.Usecase
}

func NewHandler(usecase users.Usecase) Handler {
	return Handler{usecase: usecase}
}

// GetUserData godoc
// @Summary      Return the current user's profile
// @Description  Reads the authenticated user from the JWT in the Authorization header and returns the matching record (in-memory cache first, Postgres on miss).
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  v1.BaseResponse{data=responses.UserResponse}  "User profile"
// @Failure      401  {object}  v1.BaseResponse  "Missing or invalid token"
// @Failure      404  {object}  v1.BaseResponse  "User no longer exists"
// @Router       /users/me [get]
func (h Handler) GetUserData(ctx *gin.Context) {
	user, err := httpauth.CurrentUserFromContext(ctx)
	if err != nil {
		v1.NewErrorResponse(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	userDom, err := h.usecase.GetByEmail(ctx.Request.Context(), user.Email)
	if err != nil {
		v1.RespondWithError(ctx, err)
		return
	}

	v1.NewSuccessResponse(ctx, http.StatusOK, "user data fetched successfully", map[string]interface{}{
		"user": responses.FromV1Domain(userDom),
	})
}
