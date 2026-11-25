package usecase_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Provision stores a verified user with the given roles and mails nothing, and refuses invalid input before
// the store is called and an email another user has.
func TestProvision(t *testing.T) {
	validInput := usecase.ProvisionInput{
		Email:       userEmail,
		Password:    plainPassword,
		DisplayName: "Ada",
		Roles:       []string{"admin", "editor"},
	}
	inputWith := func(change func(input *usecase.ProvisionInput)) usecase.ProvisionInput {
		input := validInput
		change(&input)
		return input
	}
	provisioned := func(t *testing.T) *domain.User {
		return userWith(t, func(stored *domain.UserStored) {
			stored.ID = registeredID
			stored.Roles = []string{"admin", "editor"}
		})
	}
	tCases := []struct {
		name    string
		input   usecase.ProvisionInput
		mock    func(f *fixture)
		want    usecase.UserView
		wantErr error
	}{
		{
			name:  "a verified user with the given roles is stored",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(plainPassword).Return(passwordHash, nil)
				f.store.users.EXPECT().Create(gomock.Any(), provisioned(f.t)).Return(nil)
			},
			want: viewWith(func(view *usecase.UserView) {
				view.ID = registeredID
				view.Roles = []string{"admin", "editor"}
			}),
		},
		{
			name:  "an email another user has is refused",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(gomock.Any()).Return(passwordHash, nil)
				f.store.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrEmailTaken)
			},
			wantErr: domain.ErrEmailTaken,
		},
		{
			name:  "a store that cannot be reached fails the call",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(gomock.Any()).Return(passwordHash, nil)
				f.store.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:    "a weak password is refused before the store is called",
			input:   inputWith(func(input *usecase.ProvisionInput) { input.Password = strings.Repeat("a", 73) }),
			mock:    func(*fixture) {},
			wantErr: domain.ErrWeakPassword,
		},
		{
			name:    "an invalid email is refused before the store is called",
			input:   inputWith(func(input *usecase.ProvisionInput) { input.Email = "ada" }),
			mock:    func(*fixture) {},
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:  "a password that cannot be hashed fails the call before the store is called",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(gomock.Any()).Return("", errHashing)
			},
			wantErr: errHashing,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Provision(t.Context(), tc.input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Provision = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Provision = %+v, want %+v", got, tc.want)
			}
		})
	}
}
