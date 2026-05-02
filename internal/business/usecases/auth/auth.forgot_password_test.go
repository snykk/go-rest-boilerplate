package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestForgotPassword(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		setup   func(f *fixture)
		wantErr bool
	}{
		{
			name:  "happy path persists token in Redis and queues email",
			email: "patrick@example.com",
			setup: func(f *fixture) {
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").Return(activeUser(t), nil).Once()
				f.redis.On("Set", mock.Anything, mock.MatchedBy(func(k string) bool {
					return len(k) > len("pwd_reset:") && k[:len("pwd_reset:")] == "pwd_reset:"
				}), "user-1").Return(nil).Once()
				f.redis.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil).Once()
				f.mailer.On("SendPasswordReset", mock.AnythingOfType("string"), "patrick@example.com").Return(nil).Once()
			},
		},
		{
			// Defeat user enumeration: unknown email returns 200 OK with no observable side-effect.
			name:  "unknown email is swallowed silently (no Set / mailer call)",
			email: "ghost@example.com",
			setup: func(f *fixture) {
				f.users.On("GetByEmail", mock.Anything, "ghost@example.com").
					Return(domain.User{}, apperror.NotFound("email not found")).Once()
			},
		},
		{
			name:  "infra error from users.GetByEmail bubbles up",
			email: "patrick@example.com",
			setup: func(f *fixture) {
				f.users.On("GetByEmail", mock.Anything, "patrick@example.com").
					Return(domain.User{}, apperror.InternalCause(errors.New("redis down"))).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)
			err := f.usecase.ForgotPassword(context.Background(), tt.email)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}
