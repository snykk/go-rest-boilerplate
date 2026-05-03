//go:build integration

package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/test/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ChangePassword_OldPasswordRejected(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	user := register(t, fix, "change@example.com", "Old_Pwd_123!")

	require.NoError(t, fix.Auth.ChangePassword(ctx, user.ID, "Old_Pwd_123!", "New_Pwd_456!"))

	// Old password no longer works.
	_, err := fix.Auth.Login(ctx, "change@example.com", "Old_Pwd_123!")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)

	// New password does.
	out, err := fix.Auth.Login(ctx, "change@example.com", "New_Pwd_456!")
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
}

func TestE2E_ChangePassword_WrongCurrentRejected(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	user := register(t, fix, "wrong@example.com", "Old_Pwd_123!")

	err := fix.Auth.ChangePassword(ctx, user.ID, "totally-wrong", "New_Pwd_456!")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}

func TestE2E_ChangePassword_RevokesPreExistingRefreshTokens(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	user := register(t, fix, "revoke@example.com", "Old_Pwd_123!")

	// Issue a refresh token before the password change.
	loginOut, err := fix.Auth.Login(ctx, "revoke@example.com", "Old_Pwd_123!")
	require.NoError(t, err)
	staleRefresh := loginOut.RefreshToken
	require.NotEmpty(t, staleRefresh)

	require.NoError(t, fix.Auth.ChangePassword(ctx, user.ID, "Old_Pwd_123!", "New_Pwd_456!"))

	// The pre-existing refresh token must be rejected — credential
	// rotation has to close active sessions.
	_, err = fix.Auth.Refresh(ctx, staleRefresh)
	require.Error(t, err, "refresh token issued before password change must be revoked")
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}

func TestE2E_ForgotPassword_ResetUnlocksLogin(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "forgot@example.com", "Old_Pwd_123!")

	require.NoError(t, fix.Auth.ForgotPassword(ctx, "forgot@example.com"))
	// The capturing mailer stores both OTP codes and reset tokens
	// in the same slice; the most recent capture for this email is
	// the reset token.
	resetToken := fix.Mailer.LastOTP(t, "forgot@example.com")
	require.NotEmpty(t, resetToken)

	require.NoError(t, fix.Auth.ResetPassword(ctx, resetToken, "Reset_Pwd_789!"))

	// Old password no longer works.
	_, err := fix.Auth.Login(ctx, "forgot@example.com", "Old_Pwd_123!")
	require.Error(t, err)

	// New password does.
	out, err := fix.Auth.Login(ctx, "forgot@example.com", "Reset_Pwd_789!")
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
}

func TestE2E_ForgotPassword_UnknownEmailNoOp(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	// Defeats user-enumeration: unknown email returns nil to the
	// caller. (No mailer call expected, but the capturing mailer is
	// permissive — we just assert no error.)
	require.NoError(t, fix.Auth.ForgotPassword(ctx, "nobody@example.com"))
}

func TestE2E_ResetPassword_TokenIsSingleUse(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "su@example.com", "Old_Pwd_123!")

	require.NoError(t, fix.Auth.ForgotPassword(ctx, "su@example.com"))
	resetToken := fix.Mailer.LastOTP(t, "su@example.com")

	require.NoError(t, fix.Auth.ResetPassword(ctx, resetToken, "First_Reset_123!"))

	// Replaying the same token must fail — Redis Del + revocation
	// cutoff combined.
	err := fix.Auth.ResetPassword(ctx, resetToken, "Second_Reset_456!")
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}
