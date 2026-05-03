package auth_test

import (
	"context"
	"errors"
	"testing"

	golangJWT "github.com/golang-jwt/jwt/v5"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func refreshClaims(jti, email string) jwt.JwtCustomClaim {
	return jwt.JwtCustomClaim{
		UserID:           "user-1",
		Email:            email,
		Kind:             jwt.KindRefresh,
		RegisteredClaims: golangJWT.RegisteredClaims{ID: jti},
	}
}

func TestRefresh(t *testing.T) {
	tests := []struct {
		name  string
		token string
		setup func(f *fixture)
		// wantErr / wantErrType paired because apperror.ErrTypeInternal
		// is the iota zero — a single sentinel would collide with that
		// type and silently pass.
		wantErr     bool
		wantErrType apperror.ErrorType
	}{
		{
			name:  "happy path mints new pair and revokes the old JTI last",
			token: "old-refresh-tok",
			setup: func(f *fixture) {
				user := activeUser(t)
				oldJTI := "old-jti"
				f.jwt.On("ParseRefreshToken", "old-refresh-tok").Return(refreshClaims(oldJTI, user.Email), nil).Once()
				f.redis.On("Get", mock.Anything, "refresh:"+oldJTI).Return(oldJTI, nil).Once()
				f.users.On("GetByEmail", mock.Anything, users.GetByEmailRequest{Email: user.Email}).Return(users.GetByEmailResponse{User: user}, nil).Once()
				f.jwt.On("GenerateTokenPair", user.ID, false, user.Email).Return(samplePair(), nil).Once()
				f.redis.On("Set", mock.Anything, "refresh:refresh-jti", "refresh-jti").Return(nil).Once()
				f.redis.On("Expire", mock.Anything, "refresh:refresh-jti", mock.AnythingOfType("time.Duration")).Return(nil).Once()
				// Old JTI deleted last (after the new one is persisted).
				f.redis.On("Del", mock.Anything, "refresh:"+oldJTI).Return(nil).Once()
			},
		},
		{
			name:  "revoked token (Redis miss on JTI) returns Unauthorized",
			token: "stale-tok",
			setup: func(f *fixture) {
				jti := "stale-jti"
				f.jwt.On("ParseRefreshToken", "stale-tok").Return(refreshClaims(jti, "x@y.com"), nil).Once()
				// Redis Get returns an error → token has been revoked.
				f.redis.On("Get", mock.Anything, "refresh:"+jti).Return("", errors.New("redis: nil")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
		},
		{
			name:  "invalid signature surfaces as Unauthorized (no Redis call)",
			token: "bogus",
			setup: func(f *fixture) {
				f.jwt.On("ParseRefreshToken", "bogus").Return(jwt.JwtCustomClaim{}, errors.New("bad signature")).Once()
			},
			wantErr:     true,
			wantErrType: apperror.ErrTypeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			out, err := f.usecase.Refresh(context.Background(), auth.RefreshRequest{RefreshToken: tt.token})

			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, "access-tok", out.AccessToken)
				assert.Equal(t, "refresh-tok", out.RefreshToken)
				return
			}
			require.Error(t, err)
			var domErr *apperror.DomainError
			require.True(t, errors.As(err, &domErr))
			assert.Equal(t, tt.wantErrType, domErr.Type)
		})
	}
}
