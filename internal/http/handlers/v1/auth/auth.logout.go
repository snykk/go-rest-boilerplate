package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// Logout godoc
// @Summary      Revoke the refresh token
// @Description  Deletes the refresh-token jti from Redis so /auth/refresh rejects it. Access tokens remain valid until their natural expiry — clients should discard them on logout.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserRefreshRequest  true  "Refresh token to revoke"
// @Success      200      {object}  v1.BaseResponse  "Logged out"
// @Failure      401      {object}  v1.BaseResponse  "Refresh token invalid"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/logout [post]
func (h Handler) Logout(ctx *gin.Context) {
	var req requests.UserRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventLogout
	ev.Success = true
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "logout success", nil)
}
