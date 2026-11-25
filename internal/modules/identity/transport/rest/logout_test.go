package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"
)

// Logout answers 204 with no body, also for a token no session has, which the use cases accept. It refuses a
// body without a token before the use cases are called.
func TestHandlerLogout(t *testing.T) {
	const body = `{"refresh_token":"` + refreshToken + `"}`
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{
			name: "the sign-in of the token is ended",
			body: body,
			mock: func(f *fixture) { f.useCases.EXPECT().Logout(gomock.Any(), refreshToken).Return(nil) },
			want: outcome{status: 204},
		},
		{
			name: "a missing token is refused before the use cases are called",
			body: `{}`,
			want: outcome{status: 422, code: "validation_failed", field: "refresh_token"},
		},
		{
			name: "a store failure is an internal error that does not leak its cause",
			body: body,
			mock: func(f *fixture) { f.useCases.EXPECT().Logout(gomock.Any(), refreshToken).Return(errStoreDown) },
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/logout", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
