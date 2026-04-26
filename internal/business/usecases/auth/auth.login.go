package auth

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
)

// Login validates credentials and returns a fresh access+refresh
// token pair. Wrong password and unknown email take the same wall
// time to mask user enumeration.
func (uc *usecase) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = normalizeEmail(email)
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		// Run a dummy bcrypt comparison so this path takes roughly
		// the same wall-clock time as a real password check. Without
		// it, an attacker can enumerate valid emails by measuring
		// response latency.
		_ = helpers.ValidateHash(password, dummyBcryptHash)
		return LoginResult{}, apperror.Unauthorized("invalid email or password")
	}

	if !user.Active {
		return LoginResult{}, apperror.Forbidden("account is not activated")
	}

	if !helpers.ValidateHash(password, user.Password) {
		return LoginResult{}, apperror.Unauthorized("invalid email or password")
	}

	isAdmin := user.RoleID == constants.AdminID
	pair, err := uc.jwtService.GenerateTokenPair(user.ID, isAdmin, user.Email)
	if err != nil {
		return LoginResult{}, apperror.InternalCause(fmt.Errorf("generate token: %w", err))
	}
	if err := uc.rememberRefresh(ctx, pair); err != nil {
		// If Redis is unavailable we'd rather fail login than issue
		// a refresh token the /refresh endpoint can't verify.
		return LoginResult{}, apperror.InternalCause(fmt.Errorf("persist refresh: %w", err))
	}

	return LoginResult{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}
