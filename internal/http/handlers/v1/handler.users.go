package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	httpauth "github.com/snykk/go-rest-boilerplate/internal/http/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
)

// UserHandler serves user-domain endpoints (/users/*). It calls into
// users.Usecase only — auth flows live on AuthHandler.
type UserHandler struct {
	usecase users.Usecase
}

func NewUserHandler(usecase users.Usecase) UserHandler {
	return UserHandler{usecase: usecase}
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
func (h UserHandler) GetUserData(ctx *gin.Context) {
	user, err := httpauth.CurrentUserFromContext(ctx)
	if err != nil {
		NewErrorResponse(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	userDom, err := h.usecase.GetByEmail(ctx.Request.Context(), user.Email)
	if err != nil {
		RespondWithError(ctx, err)
		return
	}

	NewSuccessResponse(ctx, http.StatusOK, "user data fetched successfully", map[string]interface{}{
		"user": responses.FromV1Domain(userDom),
	})
}
