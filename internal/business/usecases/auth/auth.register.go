package auth

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

// Register creates a fresh, inactive user account. The user must
// follow up with SendOTP + VerifyOTP before Login will succeed.
//
// Pure delegation today; lives behind the auth boundary so future
// pre-checks (rate limit by IP, blocklist, captcha, invitation-code
// gates) can land here without the User context having to know
// about them.
func (uc *usecase) Register(ctx context.Context, in *entities.UserDomain) (entities.UserDomain, error) {
	return uc.users.Store(ctx, in)
}
