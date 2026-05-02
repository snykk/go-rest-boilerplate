package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByID(t *testing.T) {
	expected := sampleUser()

	tests := []struct {
		name     string
		inputID  string
		setup    func(f *fixture)
		wantUser domain.User
		// wantErr / wantErrType paired because apperror.ErrTypeInternal
		// is the iota zero — a single sentinel would collide with that
		// type and silently pass.
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:    "happy path passes repo result through unchanged",
			inputID: expected.ID,
			setup: func(f *fixture) {
				f.repo.On("GetByID", context.Background(), expected.ID).Return(expected, nil).Once()
			},
			wantUser: expected,
		},
		{
			name:    "repo NotFound preserves DomainError type",
			inputID: "missing",
			setup: func(f *fixture) {
				f.repo.On("GetByID", context.Background(), "missing").
					Return(domain.User{}, apperror.NotFound("user not found")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeNotFound,
		},
		{
			name:    "raw repo error gets wrapped into DomainError Internal",
			inputID: "any",
			setup: func(f *fixture) {
				// A plain Go error (e.g., connection refused) must be
				// upgraded into a DomainError so the HTTP layer can
				// map it to a status code.
				f.repo.On("GetByID", context.Background(), "any").
					Return(domain.User{}, errors.New("connection refused")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.GetByID(context.Background(), tt.inputID)

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
