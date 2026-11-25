package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Refresh answers 200 with the next token pair, which no cache may keep, and 401 invalid_token for a refresh
// token the use cases refuse. It refuses a body without a token before the use cases are called.
func TestHandlerRefresh(t *testing.T) {
	const body = `{"refresh_token":"` + refreshToken + `"}`
	expectRefreshFails := func(err error) func(f *fixture) {
		return func(f *fixture) {
			f.useCases.EXPECT().Refresh(gomock.Any(), refreshToken).Return(usecase.TokenPair{}, err)
		}
	}
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{
			name: "a live refresh token gets the next pair",
			body: body,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Refresh(gomock.Any(), refreshToken).Return(pairFixture(), nil)
			},
			want: outcome{status: 200, body: pairJSON, noStore: true},
		},
		{
			name: "a missing token is refused before the use cases are called",
			body: `{}`,
			want: outcome{status: 422, code: "validation_failed", field: "refresh_token"},
		},
		{
			name: "a refused token is unauthorized",
			body: body,
			mock: expectRefreshFails(domain.ErrInvalidToken),
			want: outcome{status: 401, code: "invalid_token"},
		},
		{
			name: "an unexpected error is an internal error that does not leak its cause",
			body: body,
			mock: expectRefreshFails(errStoreDown),
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/refresh", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
