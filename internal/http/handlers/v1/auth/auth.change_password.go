package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	httpauth "github.com/snykk/go-rest-boilerplate/internal/http/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// ChangePassword godoc
// @Summary      Change the authenticated user's password
// @Description  Verifies the current password, swaps it for the new one, and stamps the revocation cutoff so refresh tokens issued before now are rejected.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      requests.ChangePasswordRequest  true  "Current + new password"
// @Success      200  {object}  v1.BaseResponse  "Password changed"
// @Failure      400  {object}  v1.BaseResponse  "Malformed JSON body"
// @Failure      401  {object}  v1.BaseResponse  "Current password incorrect"
// @Failure      422  {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/password/change [put]
func (h Handler) ChangePassword(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "ChangePassword"
		fileName       = "auth.change_password.go"
	)
	current, err := httpauth.CurrentUserFromContext(ctx)
	if err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "ChangePassword: not authenticated", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	var req requests.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "ChangePassword: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "ChangePassword: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"user_id":              current.ID,
				"has_current_password": req.CurrentPassword != "",
				"has_new_password":     req.NewPassword != "",
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.ChangePassword(ctx.Request.Context(), current.ID, req.CurrentPassword, req.NewPassword); err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventPasswordChangeFail
		ev.UserID = current.ID
		ev.Email = current.Email
		ev.Reason = err.Error()
		audit.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "ChangePassword failed in controller", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"user_id":    current.ID,
		})
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventPasswordChangeOK
	ev.Success = true
	ev.UserID = current.ID
	ev.Email = current.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "password changed", nil)
}
