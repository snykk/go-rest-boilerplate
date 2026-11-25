package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Me answers 200 with the signed-in caller's profile, 404 when the caller is gone, and 401 to a request with no
// caller.
func TestHandlerMe(t *testing.T) {
	expectMeFails := func(err error) func(f *fixture) {
		return func(f *fixture) { f.useCases.EXPECT().Me(gomock.Any(), userID).Return(usecase.UserView{}, err) }
	}
	tCases := []struct {
		name   string
		target string
		mock   func(f *fixture)
		want   outcome
	}{
		{
			name:   "the caller gets their profile",
			target: "/v1/users/me",
			mock:   func(f *fixture) { f.useCases.EXPECT().Me(gomock.Any(), userID).Return(viewFixture(), nil) },
			want:   outcome{status: 200, body: userJSON},
		},
		{
			name:   "a request with no caller is unauthenticated",
			target: "/v2/users/me",
			want:   outcome{status: 401, code: "unauthenticated"},
		},
		{
			name:   "a caller who is gone is not found",
			target: "/v1/users/me",
			mock:   expectMeFails(domain.ErrNotFound),
			want:   outcome{status: 404, code: "not_found"},
		},
		{
			name:   "an unexpected error is an internal error that does not leak its cause",
			target: "/v1/users/me",
			mock:   expectMeFails(errStoreDown),
			want:   outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodGet, tc.target, "")

			f.check(t, recorder, tc.want)
		})
	}
}
