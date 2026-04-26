package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

// Login godoc
// @Summary      Authenticate and issue a token pair
// @Description  Returns an access token (short TTL) and a refresh token (longer TTL). Wrong password and unknown email take the same wall time to prevent user enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserLoginRequest  true  "Login credentials"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "Tokens issued"
// @Failure      400      {object}  v1.BaseResponse                                "Malformed JSON body"
// @Failure      401      {object}  v1.BaseResponse                                "Invalid email or password"
// @Failure      403      {object}  v1.BaseResponse                                "Account not yet activated"
// @Failure      422      {object}  v1.BaseResponse                                "Validation error"
// @Router       /auth/login [post]
func (h Handler) Login(ctx *gin.Context) {
	var req requests.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		v1.RespondWithError(ctx, err)
		return
	}

	result, err := h.usecase.Login(ctx.Request.Context(), req.Email, req.Password)
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventLoginFailure
		ev.Email = req.Email
		ev.Reason = err.Error()
		audit.Record(ev)
		v1.RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventLoginSuccess
	ev.Success = true
	ev.UserID = result.User.ID
	ev.Email = result.User.Email
	audit.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusOK, "login success", responses.FromLoginResult(result))
}
