package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	authuc "github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// Logout godoc
// @Summary      Revoke the refresh token
// @Description  Deletes the refresh-token jti from Redis so /auth/refresh rejects it. Access tokens remain valid until their natural expiry — clients should discard them on logout.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.RefreshRequest  true  "Refresh token to revoke"
// @Success      200      {object}  v1.BaseResponse  "Logged out"
// @Failure      401      {object}  v1.BaseResponse  "Refresh token invalid"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/logout [post]
func (h Handler) Logout(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "Logout"
		fileName       = "auth.logout.go"
	)
	var req requests.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Logout: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Logout: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"has_refresh_token": req.RefreshToken != "",
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.Logout(ctx.Request.Context(), authuc.LogoutRequest{RefreshToken: req.RefreshToken}); err != nil {
		logger.ErrorWithContext(ctx.Request.Context(), "Logout failed in controller", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventLogout
	ev.Success = true
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "logout success", nil)
}
