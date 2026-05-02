package domain_test

import (
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		email       string
		password    string
		bcryptCost  int
		wantErr     error
		extraAssert func(t *testing.T, u *domain.User)
	}{
		{
			name:       "happy path normalizes email + hashes password + UTC CreatedAt",
			username:   "patrick",
			email:      "  Patrick@Example.COM ",
			password:   "Pwd_123!",
			bcryptCost: bcrypt.MinCost,
			extraAssert: func(t *testing.T, u *domain.User) {
				assert.Equal(t, "patrick@example.com", u.Email)
				assert.NotEqual(t, "Pwd_123!", u.Password)
				assert.True(t, u.VerifyPassword("Pwd_123!"))
				assert.Equal(t, "UTC", u.CreatedAt.Location().String())
				assert.False(t, u.Active)
			},
		},
		{name: "empty username rejected", username: "   ", email: "x@y.com", password: "p", bcryptCost: bcrypt.MinCost, wantErr: domain.ErrEmptyUsername},
		{name: "empty password rejected", username: "u", email: "x@y.com", password: "", bcryptCost: bcrypt.MinCost, wantErr: domain.ErrEmptyPassword},
		{name: "empty email rejected", username: "u", email: "   ", password: "p", bcryptCost: bcrypt.MinCost, wantErr: domain.ErrEmptyEmail},
		{name: "malformed email rejected", username: "u", email: "not-an-email", password: "p", bcryptCost: bcrypt.MinCost, wantErr: domain.ErrInvalidEmail},
		{
			name:       "out-of-range bcrypt cost falls back to default",
			username:   "u",
			email:      "x@y.com",
			password:   "p",
			bcryptCost: 999,
			extraAssert: func(t *testing.T, u *domain.User) {
				assert.True(t, u.VerifyPassword("p"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := domain.NewUser(tt.username, tt.email, tt.password, 2, tt.bcryptCost)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "want %v, got %v", tt.wantErr, err)
				assert.Nil(t, u)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, u)
			if tt.extraAssert != nil {
				tt.extraAssert(t, u)
			}
		})
	}
}

func TestUser_Activate(t *testing.T) {
	u := &domain.User{ID: "u-1"}
	u.Activate()
	assert.True(t, u.Active)
	require.NotNil(t, u.UpdatedAt)
	assert.Equal(t, "UTC", u.UpdatedAt.Location().String())
}

func TestUser_VerifyPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	u := domain.User{Password: string(hash)}
	assert.True(t, u.VerifyPassword("secret"))
	assert.False(t, u.VerifyPassword("wrong"))
	assert.False(t, u.VerifyPassword(""))
}

func TestUser_IsAdmin(t *testing.T) {
	assert.True(t, domain.User{RoleID: domain.RoleAdmin}.IsAdmin())
	assert.False(t, domain.User{RoleID: domain.RoleUser}.IsAdmin())
	assert.False(t, domain.User{RoleID: 0}.IsAdmin())
}

func TestNormalizeEmail(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"User@Example.COM", "user@example.com"},
		{"  trim@me.com  ", "trim@me.com"},
		{"already@lower.com", "already@lower.com"},
		{"   ", ""},
	} {
		assert.Equal(t, tt.want, domain.NormalizeEmail(tt.in))
	}
}
