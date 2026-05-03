package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

// Login validates credentials and returns a fresh access+refresh
// token pair. Wrong password and unknown email take the same wall
// time to mask user enumeration.
func (uc *usecase) Login(ctx context.Context, email, password string) (out LoginResult, err error) {
	const (
		usecaseName = "auth"
		funcName    = "Login"
		fileName    = "auth.login.go"
	)
	startTime := time.Now()
	email = domain.NormalizeEmail(email)

	logger.InfoWithContext(ctx, fmt.Sprintf("Upper %s", funcName), logger.Fields{
		"usecase": usecaseName,
		"method":  funcName,
		"file":    fileName,
		"request": logger.Fields{
			"email": email,
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
			fields["response"] = logger.Fields{"user_id": out.User.ID}
		}
		logger.InfoWithContext(ctx, fmt.Sprintf("Lower %s", funcName), fields)
	}()

	user, lookupErr := uc.users.GetByEmail(ctx, email)
	if lookupErr != nil {
		// Run a dummy bcrypt comparison so this path takes roughly
		// the same wall-clock time as a real password check.
		_ = helpers.ValidateHash(password, dummyBcryptHash)
		err = apperror.Unauthorized("invalid email or password")
		logger.ErrorWithContext(ctx, "Login failed: user lookup error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "get_user_by_email",
			"error":   lookupErr.Error(),
			"email":   email,
		})
		return LoginResult{}, err
	}

	if !user.Active {
		err = apperror.Forbidden("account is not activated")
		logger.ErrorWithContext(ctx, "Login failed: account not activated", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "check_active",
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	if !user.VerifyPassword(password) {
		err = apperror.Unauthorized("invalid email or password")
		logger.ErrorWithContext(ctx, "Login failed: invalid password", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "verify_password",
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	pair, mintErr := uc.jwtService.GenerateTokenPair(user.ID, user.IsAdmin(), user.Email)
	if mintErr != nil {
		err = apperror.InternalCause(fmt.Errorf("generate token: %w", mintErr))
		logger.ErrorWithContext(ctx, "Login failed: token generation error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "generate_token_pair",
			"error":   mintErr.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	if persistErr := uc.rememberRefresh(ctx, pair); persistErr != nil {
		err = apperror.InternalCause(fmt.Errorf("persist refresh: %w", persistErr))
		logger.ErrorWithContext(ctx, "Login failed: persist refresh error", logger.Fields{
			"usecase": usecaseName,
			"method":  funcName,
			"file":    fileName,
			"step":    "persist_refresh",
			"error":   persistErr.Error(),
			"user_id": user.ID,
		})
		return LoginResult{}, err
	}

	out = LoginResult{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}
	return out, nil
}
