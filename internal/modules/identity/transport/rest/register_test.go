package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Register answers 201 with the new user. It refuses a malformed or incomplete body before the use cases are
// called, answers a value the domain refuses with a validation_failed problem that names its field, and never
// repeats the password; an unexpected error's cause is logged and never returned.
func TestHandlerRegister(t *testing.T) {
	const body = `{"email":"ada@example.com","password":"` + secretPassword + `","display_name":"Ada"}`
	expectRegisterFails := func(err error) func(f *fixture) {
		return func(f *fixture) {
			f.useCases.EXPECT().Register(gomock.Any(), gomock.Any()).Return(usecase.UserView{}, err)
		}
	}
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{
			name: "a valid body registers the user",
			body: body,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Register(gomock.Any(), usecase.RegisterInput{
					Email:       userEmail,
					Password:    secretPassword,
					DisplayName: "Ada",
				}).Return(viewFixture(), nil)
			},
			want: outcome{status: 201, body: userJSON},
		},
		{
			name: "a missing display name is refused before the use cases are called",
			body: `{"email":"ada@example.com","password":"` + secretPassword + `"}`,
			want: outcome{status: 422, code: "validation_failed", field: "display_name"},
		},
		{
			name: "an unknown field is refused before the use cases are called",
			body: `{"email":"ada@example.com","password":"` + secretPassword +
				`","display_name":"Ada","roles":["editor"]}`,
			want: outcome{status: 400, code: "malformed_json"},
		},
		{
			name: "a used email is a conflict",
			body: body,
			mock: expectRegisterFails(domain.ErrEmailTaken),
			want: outcome{status: 409, code: "email_taken"},
		},
		{
			name: "an invalid email names the email field",
			body: body,
			mock: expectRegisterFails(domain.ErrInvalidEmail),
			want: outcome{status: 422, code: "validation_failed", field: "email"},
		},
		{
			name: "a weak password names the password field, and the password is never repeated",
			body: body,
			mock: expectRegisterFails(domain.ErrWeakPassword),
			want: outcome{status: 422, code: "validation_failed", field: "password"},
		},
		{
			name: "an invalid display name names its field",
			body: body,
			mock: expectRegisterFails(domain.ErrInvalidDisplayName),
			want: outcome{status: 422, code: "validation_failed", field: "display_name"},
		},
		{
			name: "an unexpected error is an internal error that does not leak its cause",
			body: body,
			mock: expectRegisterFails(errStoreDown),
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/register", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
