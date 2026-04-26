// Package entities holds the enterprise entities — the innermost
// circle of the architecture. Entities are stable across delivery /
// API versions; HTTP versioning lives at internal/http/handlers/v1/
// and friends, not here.
package entities

import "time"

// UserDomain is the User entity. It carries only fields that
// describe the user themselves — auth artifacts (access / refresh
// tokens) live on usecases.LoginResult because they are produced by
// the auth flow, not properties of the user.
type UserDomain struct {
	ID        string
	Username  string
	Email     string
	Password  string
	Active    bool
	RoleID    int
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}
