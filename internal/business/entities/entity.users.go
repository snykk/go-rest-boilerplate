// Package entities holds the enterprise entities — the innermost
// circle of the architecture. Entities are stable across delivery /
// API versions; HTTP versioning lives at internal/http/handlers/v1/
// and friends, not here.
package entities

import "time"

// UserDomain is the User entity. The Token / RefreshToken fields
// stay here for now (callers across the codebase consume them); a
// follow-up commit splits the auth artifacts into a LoginResult
// type owned by the usecase layer so the entity ends up free of
// delivery concerns.
type UserDomain struct {
	ID           string
	Username     string
	Email        string
	Password     string
	Active       bool
	Token        string // access token (TODO: move to usecases.LoginResult)
	RefreshToken string // (TODO: move to usecases.LoginResult)
	RoleID       int
	CreatedAt    time.Time
	UpdatedAt    *time.Time
	DeletedAt    *time.Time
}
