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

// Login godoc
// @Summary      Authenticate and issue a token pair
// @Description  Returns an access token (short TTL) and a refresh token (longer TTL). Wrong password and unknown email take the same wall time to prevent user enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.LoginRequest  true  "Login credentials"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "Tokens issued"
// @Failure      400      {object}  v1.BaseResponse                                "Malformed JSON body"
// @Failure      401      {object}  v1.BaseResponse                                "Invalid email or password"
// @Failure      403      {object}  v1.BaseResponse                                "Account not yet activated"
// @Failure      422      {object}  v1.BaseResponse                                "Validation error"
// @Router       /auth/login [post]
func (h Handler) Login(ctx *gin.Context) {
	const (
		controllerName = "auth"
		funcName       = "Login"
		fileName       = "auth.login.go"
	)
	var req requests.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Login: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "Login: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"email":        req.Email,
				"has_password": req.Password != "",
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	result, err := h.usecase.Login(ctx.Request.Context(), authuc.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventLoginFailure
		ev.Email = req.Email
		ev.Reason = err.Error()
		audit.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "Login failed in controller", logger.Fields{
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
	ev.Type = audit.EventLoginSuccess
	ev.Success = true
	ev.UserID = result.User.ID
	ev.Email = result.User.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "login success", responses.FromLoginResponse(result))
}
