// Package rest serves the identity use cases as a JSON API: registration and e-mail verification, sign-in
// with rotating refresh tokens, and password resets and changes under /auth, and the caller's profile at
// /users/me, with RFC 9457 problems for errors.
package rest

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

//go:generate go tool mockgen -source=usecases.go -destination=usecases_mock_test.go -package=rest_test

// UseCases is what the handlers need from the identity use cases; *usecase.UserUsecase satisfies it.
type UseCases interface {
	// Register stores a new unverified user and mails them a verification code; errors are the domain errors
	// of usecase.UserUsecase.Register.
	Register(ctx context.Context, input usecase.RegisterInput) (usecase.UserView, error)
	// ResendVerification mails a new code to the unverified user with the given email. It returns nil, and
	// mails nothing, for an email no unverified user has.
	ResendVerification(ctx context.Context, email string) error
	// Verify marks verified the email of the user who holds code; errors are the domain errors of
	// usecase.UserUsecase.Verify.
	Verify(ctx context.Context, email, code string) error
	// Login signs a user in and returns their first token pair; errors are the domain errors of
	// usecase.UserUsecase.Login.
	Login(ctx context.Context, email, password string) (usecase.TokenPair, error)
	// Refresh spends a refresh token and returns the next pair, or domain.ErrInvalidToken.
	Refresh(ctx context.Context, refreshToken string) (usecase.TokenPair, error)
	// Logout ends the sign-in a refresh token belongs to. It returns nil for a token no session has.
	Logout(ctx context.Context, refreshToken string) error
	// ForgotPassword mails a password reset token to the verified user with the given email. It returns nil,
	// and mails nothing, for an email no verified user has.
	ForgotPassword(ctx context.Context, email string) error
	// ResetPassword sets a new password with a reset token; errors are the domain errors of
	// usecase.UserUsecase.ResetPassword.
	ResetPassword(ctx context.Context, token, newPassword string) error
	// ChangePassword sets a new password for a user who gives the current one and returns a new pair; errors
	// are the domain errors of usecase.UserUsecase.ChangePassword.
	ChangePassword(ctx context.Context, input usecase.ChangePasswordInput) (usecase.TokenPair, error)
	// Me returns the user with the given ID, or domain.ErrNotFound.
	Me(ctx context.Context, userID string) (usecase.UserView, error)
}

var _ UseCases = (*usecase.UserUsecase)(nil)
