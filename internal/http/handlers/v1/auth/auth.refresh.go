package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// Refresh godoc
// @Summary      Rotate the refresh token, return a new pair
// @Description  Verifies the supplied refresh token, mints a new access+refresh pair, and revokes the old jti in Redis. Replaying an already-rotated token returns 401.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserRefreshRequest  true  "Refresh token"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "New token pair"
// @Failure      401      {object}  v1.BaseResponse  "Refresh token invalid, expired, or already revoked"
// @Failure      403      {object}  v1.BaseResponse  "Account no longer active"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/refresh [post]
func (h Handler) Refresh(ctx *gin.Context) {
	var req requests.UserRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		v1.RespondWithError(ctx, err)
		return
	}

	result, err := h.usecase.Refresh(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventRefreshFail
		ev.Reason = err.Error()
		audit.Record(ev)
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventRefreshOK
	ev.Success = true
	ev.UserID = result.User.ID
	ev.Email = result.User.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "token refreshed", responses.FromLoginResult(result))
}
