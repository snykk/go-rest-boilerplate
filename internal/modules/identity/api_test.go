package identity_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

const userID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"

// errStoreDown is a store failing on its own: a connection it cannot make.
var errStoreDown = errors.New("dial tcp: connection refused")

// User returns the user with the given ID as contract.User, and reports an unknown ID as contract.ErrNotFound.
func TestAPIUser(t *testing.T) {
	tCases := []struct {
		name    string
		view    usecase.UserView
		meErr   error
		want    contract.User
		wantErr error
	}{
		{
			name: "a user is returned with their display name",
			view: usecase.UserView{
				ID:          userID,
				Email:       "ada@example.com",
				DisplayName: "Ada",
				Roles:       []string{"reader"},
				Verified:    true,
				CreatedAt:   time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC),
			},
			want: contract.User{ID: userID, DisplayName: "Ada"},
		},
		{name: "an unknown ID is contract.ErrNotFound", meErr: domain.ErrNotFound, wantErr: contract.ErrNotFound},
		{name: "a store failure passes through", meErr: errStoreDown, wantErr: errStoreDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			users := NewMockUserReader(gomock.NewController(t))
			users.EXPECT().Me(gomock.Any(), userID).Return(tc.view, tc.meErr)

			got, err := identity.NewAPI(users).User(t.Context(), userID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("User = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("User = %+v, want %+v", got, tc.want)
			}
		})
	}
}
