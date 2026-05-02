package auth_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/test/mocks"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"golang.org/x/crypto/bcrypt"
)

// fixture is the per-test wiring for the auth package. Each sub-test
// builds a fresh set of mocks via newFixture() so there's no shared
// state between tests.
type fixture struct {
	usecase auth.Usecase
	users   *mocks.UsersUsecase
	jwt     *mocks.JWTService
	mailer  *mocks.OTPMailer
	redis   *mocks.RedisCache
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	usersUC := mocks.NewUsersUsecase(t)
	jwtSvc := mocks.NewJWTService(t)
	otpMailer := mocks.NewOTPMailer(t)
	redis := mocks.NewRedisCache(t)
	return &fixture{
		usecase: auth.NewUsecase(usersUC, jwtSvc, otpMailer, redis, auth.Config{
			OTPMaxAttempts:   5,
			OTPTTL:           5 * time.Minute,
			PasswordResetTTL: 30 * time.Minute,
			BcryptCost:       bcrypt.MinCost,
		}),
		users:  usersUC,
		jwt:    jwtSvc,
		mailer: otpMailer,
		redis:  redis,
	}
}

// activeUser returns a stable user record with a known plaintext
// password ("Pwd_123!") whose bcrypt hash is computed once and
// reused across tests.
func activeUser(t *testing.T) domain.User {
	t.Helper()
	hash, err := helpers.GenerateHash("Pwd_123!")
	if err != nil {
		t.Fatalf("hash sample password: %v", err)
	}
	return domain.User{
		ID:       "user-1",
		Username: "patrick",
		Email:    "patrick@example.com",
		Password: hash,
		Active:   true,
		RoleID:   2,
	}
}
