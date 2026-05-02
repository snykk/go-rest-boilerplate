package auth

import "github.com/snykk/go-rest-boilerplate/internal/business/domain"

// LoginResult bundles the user record and the freshly-minted token
// pair returned by Login / Refresh. Tokens are auth-flow artifacts,
// not user properties — that's why this type lives in the auth
// package, not on domain.User.
type LoginResult struct {
	User         domain.User
	AccessToken  string
	RefreshToken string
}
