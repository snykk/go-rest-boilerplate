package domain

import (
	"time"
)

type Player struct {
	ID                string
	InstallationID    string
	Username          string
	Email             string
	Password          string
	RoleID            int
	CreatedAt         time.Time
	UpdatedAt         *time.Time
	DeletedAt         *time.Time
	PasswordChangedAt *time.Time
}
