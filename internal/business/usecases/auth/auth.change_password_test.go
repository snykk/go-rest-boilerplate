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

func TestChangePassword(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		current     string
		newPassword string
		setup       func(f *fixture)
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:        "happy path verifies current and persists new",
			userID:      "user-1",
			current:     "Pwd_123!",
			newPassword: "Newpwd_999!",
			setup: func(f *fixture) {
				f.users.On("GetByID", mock.Anything, users.GetByIDRequest{ID: "user-1"}).Return(users.GetByIDResponse{User: activeUser(t)}, nil).Once()
				f.users.On("UpdatePassword", mock.Anything, mock.MatchedBy(func(req users.UpdatePasswordRequest) bool {
					u := req.User
					return u.ID == "user-1" && u.Password != "Newpwd_999!" && u.PasswordChangedAt != nil
				})).Return(nil).Once()
			},
		},
		{
			name:        "wrong current password rejected as Unauthorized",
			userID:      "user-1",
			current:     "wrong",
			newPassword: "Newpwd_999!",
			setup: func(f *fixture) {
				f.users.On("GetByID", mock.Anything, users.GetByIDRequest{ID: "user-1"}).Return(users.GetByIDResponse{User: activeUser(t)}, nil).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
		},
		{
			name:        "empty new password rejected as BadRequest",
			userID:      "user-1",
			current:     "Pwd_123!",
			newPassword: "",
			setup:       func(f *fixture) {},
			wantErr:     true,
			wantErrType: apperror.ErrTypeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)
			err := f.usecase.ChangePassword(context.Background(), auth.ChangePasswordRequest{
				UserID:          tt.userID,
				CurrentPassword: tt.current,
				NewPassword:     tt.newPassword,
			})
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
