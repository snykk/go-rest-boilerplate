package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Me returns the view of the user with the given ID, with no password hash in it, and the store's error when
// it cannot.
func TestMe(t *testing.T) {
	tCases := []struct {
		name    string
		mock    func(f *fixture)
		want    usecase.UserView
		wantErr error
	}{
		{
			name: "a verified user is returned as a view",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
			},
			want: viewFixture(),
		},
		{
			name: "an unverified user is returned as not verified",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(unverifiedFixture(f.t), nil)
			},
			want: viewWith(func(view *usecase.UserView) { view.Verified = false }),
		},
		{
			name: "an unknown user is not found",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "a store that cannot be reached fails the call",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Me(t.Context(), userID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Me = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Me = %+v, want %+v", got, tc.want)
			}
		})
	}
}
