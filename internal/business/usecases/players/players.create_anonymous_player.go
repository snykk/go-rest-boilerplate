package players

import (
	"time"
	"context"

	// "github.com/go-rest-boilerplate/internal/apperror"
	"github.com/go-rest-boilerplate/internal/business/domain"
	// "github.com/go-rest-boilerplate/pkg/logger"
)

func (uc *usecase) CreateAnonymousPlayer(ctx context.Context, req CreateAnonymousPlayerRequest) (resp CreateAnonymousPlayerResponse, err error) {
	const (
		usecaseName = "players"
		funcName    = "CreateAnonymousPlayer"
		fileName    = "players.create_anonymous_player.go"
	)

	return CreateAnonymousPlayerResponse{
	    Player: domain.Player{
	        ID:             "string",
	        InstallationID: req.InstallationID,
	        Username:       "string",
	        Email:          "string",
	        Password:       "string",
	        RoleID:         1,
	        CreatedAt:      time.Now(),
	        UpdatedAt:      nil,
	        DeletedAt:      nil,
	        PasswordChangedAt: nil,
	    },
	}, nil
}
