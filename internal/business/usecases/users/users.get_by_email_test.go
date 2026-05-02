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

func TestGetByEmail(t *testing.T) {
	cached := sampleUser()

	tests := []struct {
		name       string
		inputEmail string
		setup      func(f *fixture)
		wantUser   domain.User // zero value = no positive identity check
		// wantErr / wantErrType: paired flag + value because
		// apperror.ErrTypeInternal is the iota zero, so a single
		// "0 means no error" sentinel would collide with that type.
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:       "cache hit returns immediately without touching repo",
			inputEmail: "patrick@example.com",
			setup: func(f *fixture) {
				f.rc.On("Get", "user/patrick@example.com").Return(cached).Once()
			},
			wantUser: cached,
		},
		{
			name:       "cache miss reads repo and populates cache",
			inputEmail: "patrick@example.com",
			setup: func(f *fixture) {
				f.rc.On("Get", "user/patrick@example.com").Return(nil).Once()
				f.repo.On("GetByEmail", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(cached, nil).Once()
				f.rc.On("Set", "user/patrick@example.com", cached).Once()
			},
			wantUser: cached,
		},
		{
			name:       "repo NotFound surfaces as DomainError NotFound",
			inputEmail: "ghost@example.com",
			setup: func(f *fixture) {
				f.rc.On("Get", "user/ghost@example.com").Return(nil).Once()
				f.repo.On("GetByEmail", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(domain.User{}, apperror.NotFound("user not found")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeNotFound,
		},
		{
			// Regression: raw repo errors must not be rewritten to NotFound.
			name:       "raw repo error surfaces as Internal, not 404",
			inputEmail: "patrick@example.com",
			setup: func(f *fixture) {
				f.rc.On("Get", "user/patrick@example.com").Return(nil).Once()
				f.repo.On("GetByEmail", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(domain.User{}, errors.New("connection refused")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeInternal,
		},
		{
			name:       "input email is normalized (trim + lowercase) before cache + repo lookups",
			inputEmail: "  Patrick@Example.COM ",
			setup: func(f *fixture) {
				// Mixed-case input must hash to the same lowercase
				// cache key as the canonical form — otherwise two
				// users with the same email-up-to-case would diverge
				// in the cache.
				f.rc.On("Get", "user/patrick@example.com").Return(cached).Once()
			},
			wantUser: cached,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.GetByEmail(context.Background(), tt.inputEmail)

			if !tt.wantErr {
				require.NoError(t, err)
				if tt.wantUser.ID != "" {
					assert.Equal(t, tt.wantUser, out)
				}
				return
			}
			require.Error(t, err)
			var domErr *apperror.DomainError
			require.True(t, errors.As(err, &domErr))
			assert.Equal(t, tt.wantErrType, domErr.Type)
		})
	}
}
