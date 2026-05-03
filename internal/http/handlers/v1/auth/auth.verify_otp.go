package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// VerifyOTP godoc
// @Summary      Verify an OTP code and activate the account
// @Description  Validates the supplied code against Redis and flips the user's active flag to true on success. Brute-force-guarded — after OTP_MAX_ATTEMPTS failures (default 5) the email is locked out for the OTP TTL window even with the correct code.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.VerifyOTPRequest  true  "Email + OTP code"
// @Success      200      {object}  v1.BaseResponse  "Account activated"
// @Failure      400      {object}  v1.BaseResponse  "Invalid OTP code"
// @Failure      403      {object}  v1.BaseResponse  "Locked out — too many invalid attempts"
// @Failure      404      {object}  v1.BaseResponse  "Email not registered"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/verify-otp [post]
func (h Handler) VerifyOTP(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "VerifyOTP"
		fileName       = "auth.verify_otp.go"
	)
	var req requests.VerifyOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "VerifyOTP: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "VerifyOTP: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"email":    req.Email,
				"has_code": req.Code != "",
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	if err := h.usecase.VerifyOTP(ctx.Request.Context(), req.Email, req.Code); err != nil {
		ev := auditFromGin(ctx)
		ev.Email = req.Email
		ev.Reason = err.Error()
		// Distinguish "you got it wrong" from "we locked the account"
		// — rate-limit + alerting want different signals.
		var domErr *apperror.DomainError
		if errors.As(err, &domErr) && domErr.Type == apperror.ErrTypeForbidden {
			ev.Type = audit.EventOTPLockout
		} else {
			ev.Type = audit.EventOTPVerifyFail
		}
		audit.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "VerifyOTP failed in controller", logger.Fields{
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
	ev.Type = audit.EventOTPVerifyOK
	ev.Success = true
	ev.Email = req.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "otp verification success", nil)
}
