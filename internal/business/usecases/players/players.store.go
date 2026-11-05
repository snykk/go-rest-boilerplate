package players

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/go-rest-boilerplate/internal/apperror"
	"github.com/go-rest-boilerplate/internal/business/domain"
	"github.com/go-rest-boilerplate/pkg/logger"
)

func (uc *usecase) Store(ctx context.Context, req StoreRequest) (resp StoreResponse, err error) {
	const (
		usecaseName = "players"
		funcName    = "Store"
		fileName    = "players.store.go"
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

	player, buildErr := domain.NewAnonymousPlayer(uuid.NewString(), req.InstallationId)
	if buildErr != nil {
		if errors.Is(buildErr, domain.ErrEmptyInstallationId) {
			err = apperror.BadRequest(buildErr.Error())
		} else {
			err = apperror.InternalCause(fmt.Errorf("build player: %w", buildErr))
		}
		logger.ErrorWithContext(ctx, "Store player failed: build error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "domain_new_anonymous_player",
			"error":   buildErr.Error(),
		})
		return StoreResponse{}, err
	}

	stored, repoErr := uc.repo.Store(ctx, player)
	if repoErr != nil {
		err = mapRepoError(repoErr, "store player")
		logger.ErrorWithContext(ctx, "Store player failed: repository error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "repo_store",
			"error":   repoErr.Error(),
		})
		return StoreResponse{}, err
	}
	resp = StoreResponse{Player: stored}
	return resp, nil
}