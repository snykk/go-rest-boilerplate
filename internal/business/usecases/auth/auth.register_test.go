package auth_test

import (
	"context"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	stored := domain.User{ID: "u-1", Email: "x@y.com"}

	tests := []struct {
		name    string
		in      *domain.User
		setup   func(f *fixture)
		wantOut domain.User
	}{
		{
			name: "delegates to users.Store and returns its result unchanged",
			in: &domain.User{
				Username: "x", Email: "x@y.com", Password: "Pwd_123!", RoleID: 2,
			},
			setup: func(f *fixture) {
				f.users.On("Store", mock.Anything, mock.AnythingOfType("users.StoreRequest")).
					Return(users.StoreResponse{User: stored}, nil).Once()
			},
			wantOut: stored,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.Register(context.Background(), auth.RegisterRequest{User: tt.in})
			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.User)
		})
	}
}
