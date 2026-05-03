//go:build integration

package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/test/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ChangePassword_OldPasswordRejected(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	user := register(t, fix, "change@example.com", "Old_Pwd_123!")

	require.NoError(t, fix.Auth.ChangePassword(ctx, auth.ChangePasswordRequest{
		UserID:          user.ID,
		CurrentPassword: "Old_Pwd_123!",
		NewPassword:     "New_Pwd_456!",
	}))

	_, err := fix.Auth.Login(ctx, auth.LoginRequest{Email: "change@example.com", Password: "Old_Pwd_123!"})
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)

	out, err := fix.Auth.Login(ctx, auth.LoginRequest{Email: "change@example.com", Password: "New_Pwd_456!"})
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
}

func TestE2E_ChangePassword_WrongCurrentRejected(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	user := register(t, fix, "wrong@example.com", "Old_Pwd_123!")

	err := fix.Auth.ChangePassword(ctx, auth.ChangePasswordRequest{
		UserID:          user.ID,
		CurrentPassword: "totally-wrong",
		NewPassword:     "New_Pwd_456!",
	})
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}

func TestE2E_ChangePassword_RevokesPreExistingRefreshTokens(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	user := register(t, fix, "revoke@example.com", "Old_Pwd_123!")

	loginOut, err := fix.Auth.Login(ctx, auth.LoginRequest{Email: "revoke@example.com", Password: "Old_Pwd_123!"})
	require.NoError(t, err)
	staleRefresh := loginOut.RefreshToken
	require.NotEmpty(t, staleRefresh)

	require.NoError(t, fix.Auth.ChangePassword(ctx, auth.ChangePasswordRequest{
		UserID:          user.ID,
		CurrentPassword: "Old_Pwd_123!",
		NewPassword:     "New_Pwd_456!",
	}))

	_, err = fix.Auth.Refresh(ctx, auth.RefreshRequest{RefreshToken: staleRefresh})
	require.Error(t, err, "refresh token issued before password change must be revoked")
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}

func TestE2E_ForgotPassword_ResetUnlocksLogin(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "forgot@example.com", "Old_Pwd_123!")

	require.NoError(t, fix.Auth.ForgotPassword(ctx, auth.ForgotPasswordRequest{Email: "forgot@example.com"}))
	resetToken := fix.Mailer.LastOTP(t, "forgot@example.com")
	require.NotEmpty(t, resetToken)

	require.NoError(t, fix.Auth.ResetPassword(ctx, auth.ResetPasswordRequest{Token: resetToken, NewPassword: "Reset_Pwd_789!"}))

	_, err := fix.Auth.Login(ctx, auth.LoginRequest{Email: "forgot@example.com", Password: "Old_Pwd_123!"})
	require.Error(t, err)

	out, err := fix.Auth.Login(ctx, auth.LoginRequest{Email: "forgot@example.com", Password: "Reset_Pwd_789!"})
	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
}

func TestE2E_ForgotPassword_UnknownEmailNoOp(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	require.NoError(t, fix.Auth.ForgotPassword(ctx, auth.ForgotPasswordRequest{Email: "nobody@example.com"}))
}

func TestE2E_ResetPassword_TokenIsSingleUse(t *testing.T) {
	fix := testenv.NewAuthFixture(t)
	ctx := context.Background()

	register(t, fix, "su@example.com", "Old_Pwd_123!")

	require.NoError(t, fix.Auth.ForgotPassword(ctx, auth.ForgotPasswordRequest{Email: "su@example.com"}))
	resetToken := fix.Mailer.LastOTP(t, "su@example.com")

	require.NoError(t, fix.Auth.ResetPassword(ctx, auth.ResetPasswordRequest{Token: resetToken, NewPassword: "First_Reset_123!"}))

	err := fix.Auth.ResetPassword(ctx, auth.ResetPasswordRequest{Token: resetToken, NewPassword: "Second_Reset_456!"})
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeUnauthorized, domErr.Type)
}
