package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	authuc "github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// SendOTP godoc
// @Summary      Send an OTP code to the user's email
// @Description  Generates a 6-digit OTP, stores it in Redis with a TTL, and enqueues the email via the async mailer. The HTTP response returns on enqueue, not on actual SMTP delivery.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.SendOTPRequest  true  "Email to send OTP to"
// @Success      200      {object}  v1.BaseResponse  "OTP enqueued"
// @Failure      404      {object}  v1.BaseResponse  "Email not registered"
// @Failure      400      {object}  v1.BaseResponse  "Account already activated"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Failure      500      {object}  v1.BaseResponse  "Failed to enqueue mail"
// @Router       /auth/send-otp [post]
func (h Handler) SendOTP(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "SendOTP"
		fileName       = "auth.send_otp.go"
	)
	var req requests.SendOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "SendOTP: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "SendOTP: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"email": req.Email,
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.SendOTP(ctx.Request.Context(), authuc.SendOTPRequest{Email: req.Email}); err != nil {
		logger.ErrorWithContext(ctx.Request.Context(), "SendOTP failed in controller", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"email":      req.Email,
		})
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventOTPSent
	ev.Success = true
	ev.Email = req.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, fmt.Sprintf("otp code has been send to %s", req.Email), nil)
}
