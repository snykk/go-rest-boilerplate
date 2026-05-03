package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	tests := []struct {
		name string
		in   *domain.User
		// setup wires the per-case mock expectations on a fresh
		// fixture. Each case gets a clean fixture; no state leaks
		// between cases.
		setup func(f *fixture)
		// wantErr toggles the assertion direction. When false the
		// test expects err == nil; when true the test expects a
		// *apperror.DomainError of wantErrType. We use a separate
		// flag rather than treating wantErrType == 0 as "no error"
		// because apperror.ErrTypeInternal is the iota zero — the
		// two sentinels would collide.
		wantErr     bool
		wantErrType apperror.ErrorType
		// extraAsserts runs after error checks pass; only invoked on
		// the happy path.
		extraAsserts func(t *testing.T, out domain.User)
	}{
		{
			name: "hashes password and normalizes email before storing",
			in: &domain.User{
				Username: "newuser",
				Email:    "  NewUser@Example.COM ",
				Password: "Plaintext_123!",
				RoleID:   2,
			},
			setup: func(f *fixture) {
				// MatchedBy enforces the post-normalization shape the
				// repo should see — anything else is a regression.
				f.repo.On("Store", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
					return u.Email == "newuser@example.com" &&
						u.Password != "Plaintext_123!" && u.Password != "" &&
						!u.CreatedAt.IsZero()
				})).Return(sampleUser(), nil).Once()
			},
			extraAsserts: func(t *testing.T, out domain.User) {
				assert.NotEmpty(t, out.ID)
			},
		},
		{
			name: "propagates DomainError type from repo (conflict)",
			in: &domain.User{
				Username: "dup", Email: "dup@example.com", Password: "Pwd_123!", RoleID: 2,
			},
			setup: func(f *fixture) {
				f.repo.On("Store", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(domain.User{}, apperror.Conflict("username or email already exists")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.Store(context.Background(), users.StoreRequest{User: tt.in})

			if !tt.wantErr {
				require.NoError(t, err)
				if tt.extraAsserts != nil {
					tt.extraAsserts(t, out.User)
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
