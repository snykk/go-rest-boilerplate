//go:build integration

package testenv

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	userspostgres "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/postgres/users"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/require"
)

// AuthFixture is the fully-wired auth slice used by end-to-end tests:
// real Postgres, real Redis, real Ristretto, real JWT — only the
// outbound SMTP mailer is faked, because we need to capture OTP codes
// to feed them back into VerifyOTP, and SMTP isn't worth running in
// CI anyway.
//
// Both bounded contexts are exposed: Auth for the auth flows being
// tested, Users for any setup / verification step that needs to read
// or mutate user records directly.
type AuthFixture struct {
	Auth   auth.Usecase
	Users  users.Usecase
	Mailer *CapturingMailer
	JWT    jwt.JWTService
}

// CapturingMailer records every OTP+receiver pair so tests can pluck
// the code out instead of guessing the 6 random digits.
type CapturingMailer struct {
	mu       sync.Mutex
	captured []otpCapture
}

type otpCapture struct{ Code, Receiver string }

// SendOTP satisfies mailer.OTPMailer. Always succeeds.
func (m *CapturingMailer) SendOTP(_ context.Context, code, receiver string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.captured = append(m.captured, otpCapture{Code: code, Receiver: receiver})
	return nil
}

func (m *CapturingMailer) SendPasswordReset(ctx context.Context, token, receiver string) error {
	return m.SendOTP(ctx, token, receiver)
}

// LastOTP returns the most recently captured OTP for receiver, or
// fails the test if none was sent.
func (m *CapturingMailer) LastOTP(t *testing.T, receiver string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.captured) - 1; i >= 0; i-- {
		if m.captured[i].Receiver == receiver {
			return m.captured[i].Code
		}
	}
	t.Fatalf("no OTP captured for %s", receiver)
	return ""
}

// NewAuthFixture wires both bounded contexts against fresh Postgres +
// Redis containers. Tunable knobs (OTP attempts, JWT secret length,
// bcrypt cost) are seeded from sane defaults — tests that need to
// vary them can override config.AppConfig directly before calling.
func NewAuthFixture(t *testing.T) *AuthFixture {
	t.Helper()
	db := StartPostgres(t)
	redis := StartRedis(t)

	if config.AppConfig.OTPMaxAttempts == 0 {
		config.AppConfig.OTPMaxAttempts = 5
	}
	if config.AppConfig.REDISExpired == 0 {
		config.AppConfig.REDISExpired = 5
	}
	if config.AppConfig.BcryptCost == 0 {
		// Lower cost in tests so register doesn't add 100ms+ per call.
		config.AppConfig.BcryptCost = 4
	}
	if config.AppConfig.JWTSecret == "" {
		config.AppConfig.JWTSecret = "integration-test-secret-thirty-two-chars!"
	}
	if config.AppConfig.JWTIssuer == "" {
		config.AppConfig.JWTIssuer = "integration-test"
	}
	if config.AppConfig.JWTExpired == 0 {
		config.AppConfig.JWTExpired = 1
	}
	if config.AppConfig.JWTRefreshExpired == 0 {
		config.AppConfig.JWTRefreshExpired = 7
	}

	ristretto, err := caches.NewRistrettoCache()
	require.NoError(t, err)

	jwtSvc := jwt.NewJWTServiceWithRefresh(
		config.AppConfig.JWTSecret,
		config.AppConfig.JWTIssuer,
		config.AppConfig.JWTExpired,
		config.AppConfig.JWTRefreshExpired,
	)

	mailer := &CapturingMailer{}
	repo := userspostgres.NewUserRepository(db)
	usersUC := users.NewUsecase(repo, ristretto, users.Config{
		BcryptCost: config.AppConfig.BcryptCost,
	})
	authUC := auth.NewUsecase(usersUC, jwtSvc, mailer, redis, auth.Config{
		OTPMaxAttempts:   5,
		OTPTTL:           5 * time.Minute,
		PasswordResetTTL: 30 * time.Minute,
		BcryptCost:       config.AppConfig.BcryptCost,
		LoginMaxAttempts:  10,
		LoginLockoutTTL:   15 * time.Minute,
		ForgotMaxAttempts: 3,
		ForgotLockoutTTL:  15 * time.Minute,
	})

	return &AuthFixture{
		Auth:   authUC,
		Users:  usersUC,
		Mailer: mailer,
		JWT:    jwtSvc,
	}
}
