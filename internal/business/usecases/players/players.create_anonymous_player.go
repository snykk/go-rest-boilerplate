package players

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rest-boilerplate/internal/apperror"
	"github.com/go-rest-boilerplate/pkg/logger"
)

// CreateAnonymousPlayer registers a player for the given installation
// id and mints an access token whose subject is the new player's UUID.
// Anonymous players have no e-mail, so the token's Email claim is empty
// and no refresh token is issued.
func (uc *usecase) CreateAnonymousPlayer(ctx context.Context, req CreateAnonymousPlayerRequest) (resp CreateAnonymousPlayerResponse, err error) {
	const (
		usecaseName = "players"
		funcName    = "CreateAnonymousPlayer"
		fileName    = "players.create_anonymous_player.go"
	)
	startTime := time.Now()

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"has_installation_id": req.InstallationId != "",
		},
	})

	defer func() {
		fields := logger.Fields{
			"usecase":  usecaseName,
			"method":   funcName,
			"file":     fileName,
			"duration": time.Since(startTime).Milliseconds(),
		}
		if err == nil {
			fields["response"] = logger.Fields{"player_id": resp.Player.Id}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	storeResp, storeErr := uc.Store(ctx, StoreRequest{InstallationId: req.InstallationId})
	if storeErr != nil {
		err = storeErr
		logger.ErrorWithContext(ctx, "Create anonymous player failed: store error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "players_store",
			"error":   storeErr.Error(),
		})
		return CreateAnonymousPlayerResponse{}, err
	}
	player := storeResp.Player

	token, mintErr := uc.jwtService.GenerateToken(player.Id, false, "")
	if mintErr != nil {
		err = apperror.InternalCause(fmt.Errorf("generate token: %w", mintErr))
		logger.ErrorWithContext(ctx, "Create anonymous player failed: token generation error", logger.Fields{
			"usecase":   usecaseName,
			"method":    funcName,
			"file":      fileName,
			"step":      "generate_token",
			"error":     mintErr.Error(),
			"player_id": player.Id,
		})
		return CreateAnonymousPlayerResponse{}, err
	}

	resp = CreateAnonymousPlayerResponse{Player: player, AccessToken: token}
	return resp, nil
}