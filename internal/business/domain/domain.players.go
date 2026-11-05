package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	RoleAdmin = 1
	RoleUser  = 2
)


var (
	ErrEmptyInstallationId = errors.New("installation id cannot be empty")
)

type Player struct {
	Id                string
	InstallationId    string
	Email             string
	Password          string
	RoleId            int
	CreatedAt         time.Time
	UpdatedAt         *time.Time
	DeletedAt         *time.Time
	PasswordChangedAt *time.Time
}

func NewAnonymousPlayer(id, installationID string) (*Player, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return nil, ErrEmptyInstallationId
	}
	return &Player{
		Id:             id,
		InstallationId: installationID,
		CreatedAt:      time.Now().UTC(),
	}, nil
}