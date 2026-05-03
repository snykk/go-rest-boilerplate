package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	authuc "github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// Refresh godoc
// @Summary      Rotate the refresh token, return a new pair
// @Description  Verifies the supplied refresh token, mints a new access+refresh pair, and revokes the old jti in Redis. Replaying an already-rotated token returns 401.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.RefreshRequest  true  "Refresh token"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "New token pair"
// @Failure      401      {object}  v1.BaseResponse  "Refresh token invalid, expired, or already revoked"
// @Failure      403      {object}  v1.BaseResponse  "Account no longer active"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/refresh [post]
func (h Handler) Refresh(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "Refresh"
		fileName       = "auth.refresh.go"
	)
	var req requests.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Refresh: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Refresh: validation error", logger.Fields{
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

	result, err := h.usecase.Refresh(ctx.Request.Context(), authuc.RefreshRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventRefreshFail
		ev.Reason = err.Error()
		audit.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "Refresh failed in controller", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventRefreshOK
	ev.Success = true
	ev.UserID = result.User.ID
	ev.Email = result.User.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "token refreshed", responses.FromLoginResponse(result))
}
