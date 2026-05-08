package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResetPassword(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		newPassword string
		setup       func(f *fixture)
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:        "happy path resolves token, updates password, deletes token",
			token:       "valid-tok",
			newPassword: "Newpwd_999!",
			setup: func(f *fixture) {
				f.redis.On("Get", mock.Anything, "pwd_reset:valid-tok").Return("user-1", nil).Once()
				f.users.On("GetByID", mock.Anything, users.GetByIDRequest{ID: "user-1"}).Return(users.GetByIDResponse{User: activeUser(t)}, nil).Once()
				f.users.On("UpdatePassword", mock.Anything, mock.MatchedBy(func(req users.UpdatePasswordRequest) bool {
					u := req.User
					return u.ID == "user-1" && u.PasswordChangedAt != nil
				})).Return(nil).Once()
				f.redis.On("Del", mock.Anything, "pwd_reset:valid-tok").Return(nil).Once()
				f.redis.On("Del", mock.Anything, "pwd_reset_user:user-1").Return(nil).Once()
			},
		},
		{
			name:        "missing token returns BadRequest",
			token:       "",
			newPassword: "Newpwd_999!",
			setup:       func(f *fixture) {},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
		{
			name:        "empty new password returns BadRequest",
			token:       "tok",
			newPassword: "",
			setup:       func(f *fixture) {},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
		{
			name:        "redis miss surfaces as Unauthorized",
			token:       "stale",
			newPassword: "Newpwd_999!",
			setup: func(f *fixture) {
				f.redis.On("Get", mock.Anything, "pwd_reset:stale").Return("", errors.New("redis: nil")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)
			err := f.usecase.ResetPassword(context.Background(), auth.ResetPasswordRequest{Token: tt.token, NewPassword: tt.newPassword})
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var domErr *apperror.DomainError
			require.True(t, errors.As(err, &domErr))
			assert.Equal(t, tt.wantErrType, domErr.Type)
		})
	}
}
