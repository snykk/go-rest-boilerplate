package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// ResetPassword godoc
// @Summary      Consume a reset token and set a new password
// @Description  Validates the token issued by ForgotPassword, sets the new password, and advances the revocation cutoff.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.ResetPasswordRequest  true  "Reset token + new password"
// @Success      200  {object}  v1.BaseResponse  "Password reset"
// @Failure      400  {object}  v1.BaseResponse  "Malformed JSON body"
// @Failure      401  {object}  v1.BaseResponse  "Reset token invalid or expired"
// @Failure      422  {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/password/reset [post]
func (h Handler) ResetPassword(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "ResetPassword"
		fileName       = "auth.reset_password.go"
	)
	var req requests.ResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "ResetPassword: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "ResetPassword: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"has_token":        req.Token != "",
				"has_new_password": req.NewPassword != "",
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.ResetPassword(ctx.Request.Context(), req.Token, req.NewPassword); err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventPasswordResetFail
		ev.Reason = err.Error()
		audit.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "ResetPassword failed in controller", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventPasswordResetOK
	ev.Success = true
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "password reset", nil)
}
