package auth

import "github.com/snykk/go-rest-boilerplate/internal/business/entities"

// LoginResult bundles the user record and the freshly-minted token
// pair returned by Login / Refresh. Tokens are auth-flow artifacts,
// not user properties — that's why this type lives in the auth
// package, not on entities.UserDomain.
type LoginResult struct {
	User         entities.UserDomain
	AccessToken  string
	RefreshToken string
}
