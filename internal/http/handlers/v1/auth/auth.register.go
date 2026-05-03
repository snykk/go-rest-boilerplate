package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user account in inactive state. The caller must follow up with /auth/send-otp + /auth/verify-otp to activate the account before /auth/login will succeed.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.RegisterRequest  true  "Registration payload"
// @Success      201      {object}  v1.BaseResponse{data=responses.UserResponse}  "User created"
// @Failure      400      {object}  v1.BaseResponse                                "Malformed JSON body"
// @Failure      409      {object}  v1.BaseResponse                                "Email or username already in use"
// @Failure      422      {object}  v1.BaseResponse                                "Validation error (per-field detail in data.errors)"
// @Failure      500      {object}  v1.BaseResponse                                "Internal error"
// @Router       /auth/register [post]
func (h Handler) Register(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "Register"
		fileName       = "auth.register.go"
	)
	var req requests.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Register: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Register: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"username":     req.Username,
				"email":        req.Email,
				"has_password": req.Password != "",
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	user, err := h.usecase.Register(ctx.Request.Context(), req.ToV1Domain())
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventRegister
		ev.Email = req.Email
		ev.Reason = err.Error()
		audit.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "Register failed in controller", logger.Fields{
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
	ev.Type = audit.EventRegister
	ev.Success = true
	ev.UserID = user.ID
	ev.Email = user.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusCreated, "registration user success", map[string]interface{}{
		"user": responses.FromV1Domain(user),
	})
}
