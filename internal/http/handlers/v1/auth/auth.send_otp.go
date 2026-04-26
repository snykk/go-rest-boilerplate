package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// SendOTP godoc
// @Summary      Send an OTP code to the user's email
// @Description  Generates a 6-digit OTP, stores it in Redis with a TTL, and enqueues the email via the async mailer. The HTTP response returns on enqueue, not on actual SMTP delivery.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserSendOTPRequest  true  "Email to send OTP to"
// @Success      200      {object}  v1.BaseResponse  "OTP enqueued"
// @Failure      404      {object}  v1.BaseResponse  "Email not registered"
// @Failure      400      {object}  v1.BaseResponse  "Account already activated"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Failure      500      {object}  v1.BaseResponse  "Failed to enqueue mail"
// @Router       /auth/send-otp [post]
func (h Handler) SendOTP(ctx *gin.Context) {
	var req requests.UserSendOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.SendOTP(ctx.Request.Context(), req.Email); err != nil {
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
