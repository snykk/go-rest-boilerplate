package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ResetPassword answers 204 with no body when the token sets the password. A token the use cases refuse is a
// value of the body, so it answers 422 invalid_token rather than 401; a weak password is a validation_failed
// problem that names the password field without repeating it. It refuses a body without a token before the
// use cases are called.
func TestHandlerResetPassword(t *testing.T) {
	const body = `{"token":"` + resetToken + `","password":"` + secretPassword + `"}`
	expectReset := func(err error) func(f *fixture) {
		return func(f *fixture) {
			f.useCases.EXPECT().ResetPassword(gomock.Any(), resetToken, secretPassword).Return(err)
		}
	}
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{name: "a usable token sets the password", body: body, mock: expectReset(nil), want: outcome{status: 204}},
		{
			name: "a missing token is refused before the use cases are called",
			body: `{"password":"` + secretPassword + `"}`,
			want: outcome{status: 422, code: "validation_failed", field: "token"},
		},
		{
			name: "a refused token is unprocessable",
			body: body,
			mock: expectReset(domain.ErrInvalidToken),
			want: outcome{status: 422, code: "invalid_token"},
		},
		{
			name: "a weak password names the password field, and the password is never repeated",
			body: body,
			mock: expectReset(domain.ErrWeakPassword),
			want: outcome{status: 422, code: "validation_failed", field: "password"},
		},
		{
			name: "an unexpected error is an internal error that does not leak its cause",
			body: body,
			mock: expectReset(errStoreDown),
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/password/reset", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
