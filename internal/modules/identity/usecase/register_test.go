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

// Register stores an unverified reader and its verification code in one unit of work, mails the code once the
// unit of work has committed, and refuses invalid input before anything is stored.
func TestRegister(t *testing.T) {
	validInput := usecase.RegisterInput{
		Email:       userEmail,
		Password:    plainPassword,
		DisplayName: "Ada",
	}
	inputWith := func(change func(input *usecase.RegisterInput)) usecase.RegisterInput {
		input := validInput
		change(&input)
		return input
	}
	registeredView := viewWith(func(view *usecase.UserView) {
		view.ID = registeredID
		view.Verified = false
	})
	tCases := []struct {
		name    string
		input   usecase.RegisterInput
		mock    func(f *fixture)
		want    usecase.UserView
		wantErr error
	}{
		{
			name:  "the user and its code are written in one unit of work, then mailed",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(plainPassword).Return(passwordHash, nil)
				work := f.expectUnitOfWork()
				create := f.tx.users.EXPECT().Create(gomock.Any(), registeredFixture(f.t)).Return(nil)
				put := f.tx.verifications.EXPECT().Put(gomock.Any(), codeFor(registeredID), settings().ResendCooldown).
					Return(true, nil)
				send := f.expectMailAfterCommit()
				gomock.InOrder(work, create, put, send)
			},
			want: registeredView,
		},
		{
			name: "the email is normalized and the display name trimmed",
			input: inputWith(func(input *usecase.RegisterInput) {
				input.Email = "  Ada@Example.COM "
				input.DisplayName = "  Ada "
			}),
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(plainPassword).Return(passwordHash, nil)
				f.expectUnitOfWork()
				f.tx.users.EXPECT().Create(gomock.Any(), registeredFixture(f.t)).Return(nil)
				f.tx.verifications.EXPECT().Put(gomock.Any(), codeFor(registeredID), settings().ResendCooldown).
					Return(true, nil)
				f.expectMailAfterCommit()
			},
			want: registeredView,
		},
		{
			name:  "a taken email mails nothing",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(gomock.Any()).Return(passwordHash, nil)
				f.expectUnitOfWork()
				f.tx.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrEmailTaken)
			},
			wantErr: domain.ErrEmailTaken,
		},
		{
			name:  "a code that cannot be stored fails the call and mails nothing",
			input: validInput,
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(gomock.Any()).Return(passwordHash, nil)
				f.expectUnitOfWork()
				f.tx.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				f.tx.verifications.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:    "a 73-byte password is refused before the store is called",
			input:   inputWith(func(input *usecase.RegisterInput) { input.Password = strings.Repeat("a", 73) }),
			mock:    func(*fixture) {},
			wantErr: domain.ErrWeakPassword,
		},
		{
			name:    "an invalid email is refused before the password is hashed",
			input:   inputWith(func(input *usecase.RegisterInput) { input.Email = "ada" }),
			mock:    func(*fixture) {},
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:  "an invalid display name is refused before the store is called",
			input: inputWith(func(input *usecase.RegisterInput) { input.DisplayName = " " }),
			mock: func(f *fixture) {
				f.hasher.EXPECT().Hash(gomock.Any()).Return(passwordHash, nil)
			},
			wantErr: domain.ErrInvalidDisplayName,
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

			got, err := f.useCase.Register(t.Context(), tc.input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Register = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Register = %+v, want %+v", got, tc.want)
			}
		})
	}
}
