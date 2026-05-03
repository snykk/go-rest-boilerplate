package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Register creates a fresh, inactive user account. The user must
// follow up with SendOTP + VerifyOTP before Login will succeed.
//
// Pure delegation today; lives behind the auth boundary so future
// pre-checks (rate limit by IP, blocklist, captcha, invitation-code
// gates) can land here without the User context having to know
// about them.
func (uc *usecase) Register(ctx context.Context, in *domain.User) (out domain.User, err error) {
	const (
		usecaseName = "auth"
		funcName    = "Register"
		fileName    = "auth.register.go"
	)
	startTime := time.Now()

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"username": in.Username,
			"email":    in.Email,
			"role_id":  in.RoleID,
		},
	})

	defer func() {
		duration := time.Since(startTime)
		fields := logger.Fields{
			"usecase":  usecaseName,
			"method":   funcName,
			"file":     fileName,
			"duration": duration.Milliseconds(),
		}
		if err == nil {
			fields["response"] = logger.Fields{"user_id": out.ID, "email": out.Email}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	out, err = uc.users.Store(ctx, in)
	if err != nil {
		logger.ErrorWithContext(ctx, "Register failed: store error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "users_store",
			"error":   err.Error(),
			"email":   in.Email,
		})
		return domain.User{}, err
	}
	return out, nil
}
