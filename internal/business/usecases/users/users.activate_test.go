package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestActivate(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		setup  func(f *fixture)
		// wantErr / wantErrType paired because apperror.ErrTypeInternal
		// is the iota zero — a single sentinel would collide with that
		// type and silently pass.
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:   "flips active flag with the right ID",
			userID: "user-123",
			setup: func(f *fixture) {
				// MatchedBy enforces that Activate sends the right
				// ID and Active=true to the repo. A regression that
				// e.g. forgets to set Active or passes the wrong ID
				// would not match this predicate.
				f.repo.On("ChangeActiveUser", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
					return u.ID == "user-123" && u.Active == true
				})).Return(nil).Once()
			},
		},
		{
			name:   "raw repo error becomes DomainError Internal",
			userID: "user-123",
			setup: func(f *fixture) {
				f.repo.On("ChangeActiveUser", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(errors.New("deadlock")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			err := f.usecase.Activate(context.Background(), tt.userID)

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
