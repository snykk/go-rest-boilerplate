// Package contract is what other modules may know of the identity module: the view of a user its API returns,
// the error it reports for an unknown user, and the roles a user holds. It uses the standard library only, so
// depending on it ties no module to this one's insides.
package contract

import "errors"

// The roles a user may hold, as the access tokens of the identity module carry them.
const (
	// RoleReader is the role of every user who registers.
	RoleReader = "reader"
	// RoleEditor is the role of the users who manage the content of the service. No endpoint grants it: the
	// development seed data provisions an editor, and in production an operator grants it with SQL.
	RoleEditor = "editor"
)

// ErrNotFound reports that no user has the given ID.
var ErrNotFound = errors.New("user not found")

// User is a user as other modules see it.
type User struct {
	ID          string
	DisplayName string
}
