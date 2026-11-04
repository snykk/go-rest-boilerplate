package players

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/go-rest-boilerplate/internal/business/usecases/players"
	"github.com/go-rest-boilerplate/pkg/logger"
	v1 "github.com/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/go-rest-boilerplate/pkg/validators"
	httpaudit "github.com/go-rest-boilerplate/internal/http/audit"
	auditpkg "github.com/go-rest-boilerplate/pkg/audit"

)

// @Summary      Creates a user from installation ID
// @Description  Returns an access token and user_id
// @Tags         players
// @Accept       json
// @Produce      json
// @Param        request  body      requests.LoginRequest  true  "Login credentials"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "Tokens issued"
// @Failure      400      {object}  v1.BaseResponse                                "Malformed JSON body"
// @Failure      401      {object}  v1.BaseResponse                                "Invalid email or password"
// @Failure      403      {object}  v1.BaseResponse                                "Account not yet activated"
// @Failure      422      {object}  v1.BaseResponse                                "Validation error"
// @Router       /players/login [post]
func (h Handler) CreateAnonymousPlayer(ctx *gin.Context) {
	const (
		controllerName = "players"
		funcName       = "CreateAnonymousPlayer"
		fileName       = "players.create_anonymous_player.go"
	)
	var req requests.CreateAnonymousPlayerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "CreateAnonymousPlayer: invalid request body", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
		})
		v1.NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		logger.WarnWithContext(ctx.Request.Context(), "CreateAnonymousPlayer: validation error", logger.Fields{
			"controller": controllerName,
			"method":     funcName,
			"file":       fileName,
			"error":      err.Error(),
			"request": logger.Fields{
				"installation_id": req.InstallationID,
			},
		})
		v1.RespondWithError(ctx, err)
		return
	}

	resp, err := h.usecase.CreateAnonymousPlayer(ctx, players.CreateAnonymousPlayerRequest{InstallationID: req.InstallationID})
	anonymous_player := resp.Player
	if err != nil {
		ev := httpaudit.FromGin(ctx)
		ev.Type = auditpkg.EventRegister
		ev.InstallationID = req.InstallationID
		ev.Reason = err.Error()
		auditpkg.Record(ev)
		logger.ErrorWithContext(ctx.Request.Context(), "CreateAnonymousPlayer failed in controller", logger.Fields{
			"controller":	   controllerName,
			"method":	       funcName,
			"file":	           fileName,
			"error":	       err.Error(),
			"installation_id": req.InstallationID,
		})
		v1.RespondWithError(ctx, err)
		return
	}

	ev := httpaudit.FromGin(ctx)
	ev.Type = auditpkg.EventRegister
	ev.Success = true
	ev.UserID = anonymous_player.ID
	ev.InstallationID = anonymous_player.InstallationID
	auditpkg.Record(ev)

	v1.NewSuccessResponse(ctx, http.StatusCreated, "creation anonymous player success", map[string]interface{}{
		"player": responses.PlayerFromV1Domain(anonymous_player),
	})
}