package identity_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/gateway/identity"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
)

const userID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c46"

// errModuleDown is the identity module failing on its own: a store it cannot reach.
var errModuleDown = errors.New("select user: dial tcp: connection refused")

// DisplayName returns the display name of a user the identity module knows, reports one it does not know as
// domain.ErrAuthorNotFound, and passes the module's other errors through.
func TestAuthorsDisplayName(t *testing.T) {
	tCases := []struct {
		name    string
		found   contract.User
		findErr error
		want    string
		wantErr error
	}{
		{
			name:  "a known user's display name is returned",
			found: contract.User{ID: userID, DisplayName: "Ada"},
			want:  "Ada",
		},
		{
			name:    "an unknown user is domain.ErrAuthorNotFound",
			findErr: contract.ErrNotFound,
			wantErr: domain.ErrAuthorNotFound,
		},
		{name: "another error passes through", findErr: errModuleDown, wantErr: errModuleDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			api := NewMockAPI(gomock.NewController(t))
			api.EXPECT().User(gomock.Any(), userID).Return(tc.found, tc.findErr)

			got, err := identity.NewAuthors(api).DisplayName(t.Context(), userID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("DisplayName = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("DisplayName = %q, want %q", got, tc.want)
			}
		})
	}
}
