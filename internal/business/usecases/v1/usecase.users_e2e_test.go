//go:build integration

package v1_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/test/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// register is a small helper that gets a fresh user past the
// register + verify-OTP gate so individual tests can focus on the
// scenario under test.
func register(t *testing.T, fix *testenv.AuthFixture, email, password string) V1Domains.UserDomain {
	t.Helper()
	ctx := context.Background()

	user, err := fix.Usecase.Store(ctx, &V1Domains.UserDomain{
		Username: "user_" + email,
		Email:    email,
		Password: password,
		RoleID:   2,
	})
	require.NoError(t, err)

	require.NoError(t, fix.Usecase.SendOTP(ctx, email))
	otp := fix.Mailer.LastOTP(t, email)
	require.NoError(t, fix.Usecase.VerifyOTP(ctx, email, otp))

	return user
}

func TestE2E_HappyPath_RegisterVerifyLogin(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "happy@example.com", "Secret_123!")

	out, err := fix.Usecase.Login(ctx, &V1Domains.UserDomain{
		Email:    "happy@example.com",
		Password: "Secret_123!",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Token, "Login must return an access token")
	assert.NotEmpty(t, out.RefreshToken, "Login must return a refresh token")
}

func TestE2E_OTPBruteForceLockout(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	_, err := fix.Usecase.Store(ctx, &V1Domains.UserDomain{
		Username: "lockout",
		Email:    "lock@example.com",
		Password: "Secret_123!",
		RoleID:   2,
	})
	require.NoError(t, err)
	require.NoError(t, fix.Usecase.SendOTP(ctx, "lock@example.com"))

	// Burn through OTP_MAX_ATTEMPTS with wrong codes. Each must fail
	// with BadRequest ("invalid otp code"), not Forbidden — Forbidden
	// is reserved for the lockout itself.
	for i := 0; i < 5; i++ {
		err := fix.Usecase.VerifyOTP(ctx, "lock@example.com", "000000")
		require.Error(t, err, "attempt %d", i+1)
		var domErr *apperror.DomainError
		require.True(t, errors.As(err, &domErr))
		assert.Equal(t, apperror.ErrTypeBadRequest, domErr.Type, "attempt %d: %v", i+1, domErr)
	}

	// 6th attempt must be locked out, even with the CORRECT code —
	// that's the whole point of the brute-force guard. Without this
	// test, a regression that increments attempts only on failure
	// would silently let the attacker through with an early guess.
	correctOTP := fix.Mailer.LastOTP(t, "lock@example.com")
	err = fix.Usecase.VerifyOTP(ctx, "lock@example.com", correctOTP)
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeForbidden, domErr.Type,
		"after MAX_ATTEMPTS the correct code must also be rejected with Forbidden, got %v", domErr.Type)
}

func TestE2E_LoginRejectsInactiveUser(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	// Skip the OTP step so the user stays inactive.
	_, err := fix.Usecase.Store(ctx, &V1Domains.UserDomain{
		Username: "inactive",
		Email:    "inactive@example.com",
		Password: "Secret_123!",
		RoleID:   2,
	})
	require.NoError(t, err)

	_, err = fix.Usecase.Login(ctx, &V1Domains.UserDomain{
		Email:    "inactive@example.com",
		Password: "Secret_123!",
	})
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeForbidden, domErr.Type)
}

func TestE2E_RefreshRotatesAndRevokesOldToken(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "rotate@example.com", "Secret_123!")
	loggedIn, err := fix.Usecase.Login(ctx, &V1Domains.UserDomain{
		Email:    "rotate@example.com",
		Password: "Secret_123!",
	})
	require.NoError(t, err)

	// Use the refresh token once.
	refreshed, err := fix.Usecase.Refresh(ctx, loggedIn.RefreshToken)
	require.NoError(t, err)
	assert.NotEqual(t, loggedIn.RefreshToken, refreshed.RefreshToken,
		"Refresh must rotate the token")

	// Replay the OLD refresh token — must fail because rotation
	// deleted its jti from Redis. Without this assertion, a stolen
	// refresh token could be used indefinitely.
	_, err = fix.Usecase.Refresh(ctx, loggedIn.RefreshToken)
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)

	// New refresh token must still work.
	_, err = fix.Usecase.Refresh(ctx, refreshed.RefreshToken)
	require.NoError(t, err)
}

func TestE2E_LogoutRevokesRefreshToken(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "logout@example.com", "Secret_123!")
	loggedIn, err := fix.Usecase.Login(ctx, &V1Domains.UserDomain{
		Email:    "logout@example.com",
		Password: "Secret_123!",
	})
	require.NoError(t, err)

	require.NoError(t, fix.Usecase.Logout(ctx, loggedIn.RefreshToken))

	// After logout, the refresh token must be useless.
	_, err = fix.Usecase.Refresh(ctx, loggedIn.RefreshToken)
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}

func TestE2E_VerifyOTPActivatesUserAndAllowsLogin(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	// Pre-condition: user exists but inactive — login fails.
	_, err := fix.Usecase.Store(ctx, &V1Domains.UserDomain{
		Username: "activate",
		Email:    "activate@example.com",
		Password: "Secret_123!",
		RoleID:   2,
	})
	require.NoError(t, err)

	_, err = fix.Usecase.Login(ctx, &V1Domains.UserDomain{
		Email:    "activate@example.com",
		Password: "Secret_123!",
	})
	require.Error(t, err, "login must fail before OTP verification")

	// Run the OTP flow.
	require.NoError(t, fix.Usecase.SendOTP(ctx, "activate@example.com"))
	otp := fix.Mailer.LastOTP(t, "activate@example.com")
	require.NoError(t, fix.Usecase.VerifyOTP(ctx, "activate@example.com", otp))

	// Now login must succeed — proving VerifyOTP actually flipped the
	// active flag in Postgres, not just deleted the OTP key from Redis.
	out, err := fix.Usecase.Login(ctx, &V1Domains.UserDomain{
		Email:    "activate@example.com",
		Password: "Secret_123!",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Token)
}
