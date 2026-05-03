package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Register creates a fresh, inactive user account. The user must
// follow up with SendOTP + VerifyOTP before Login will succeed.
//
// Pure delegation today; lives behind the auth boundary so future
// pre-checks (rate limit by IP, blocklist, captcha, invitation-code
// gates) can land here without the User context having to know
// about them.
func (uc *usecase) Register(ctx context.Context, req RegisterRequest) (resp RegisterResponse, err error) {
	const (
		usecaseName = "auth"
		funcName    = "Register"
		fileName    = "auth.register.go"
	)
	startTime := time.Now()
	in := req.User

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
			fields["response"] = logger.Fields{"user_id": resp.User.ID, "email": resp.User.Email}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	storeResp, storeErr := uc.users.Store(ctx, users.StoreRequest{User: in})
	if storeErr != nil {
		err = storeErr
		logger.ErrorWithContext(ctx, "Register failed: store error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "users_store",
			"error":   err.Error(),
			"email":   in.Email,
		})
		return RegisterResponse{}, err
	}
	resp = RegisterResponse{User: storeResp.User}
	return resp, nil
}
