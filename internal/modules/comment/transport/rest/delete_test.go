package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
)

// Delete answers 204 with no body when the use cases let the caller delete the comment, passing on who calls and
// whether they are an editor. It answers 403 when they refuse, 404 for an unknown comment or an ID that is not
// a UUID, and 401 to a request with no caller.
func TestHandlerDelete(t *testing.T) {
	expectDelete := func(input usecase.DeleteInput, err error) func(f *fixture) {
		return func(f *fixture) { f.useCases.EXPECT().Delete(gomock.Any(), input).Return(err) }
	}
	byReader := usecase.DeleteInput{ID: commentID, CallerID: readerID}
	tCases := []struct {
		name   string
		caller *auth.Principal
		target string // "" means /v1/comments/{commentID}
		mock   func(f *fixture)
		want   outcome
	}{
		{
			name:   "a reader's deletion is passed on as a reader's",
			caller: reader(),
			mock:   expectDelete(byReader, nil),
			want:   outcome{status: 204},
		},
		{
			name:   "an editor's deletion is passed on as an editor's",
			caller: editor(),
			mock: expectDelete(usecase.DeleteInput{
				ID:             commentID,
				CallerID:       editorID,
				CallerIsEditor: true,
			}, nil),
			want: outcome{status: 204},
		},
		{
			name:   "a caller the use cases refuse is forbidden",
			caller: reader(),
			mock:   expectDelete(byReader, domain.ErrForbidden),
			want:   outcome{status: 403, code: "forbidden"},
		},
		{
			name:   "an unknown comment is not found",
			caller: reader(),
			mock:   expectDelete(byReader, domain.ErrNotFound),
			want:   outcome{status: 404, code: "not_found"},
		},
		{
			name:   "an ID that is not a UUID is not found without asking the use cases",
			caller: reader(),
			target: "/v1/comments/not-a-uuid",
			want:   outcome{status: 404, code: "not_found"},
		},
		{
			name: "a request with no caller is unauthenticated",
			want: outcome{status: 401, code: "unauthenticated"},
		},
		{
			name:   "an unexpected error is an internal error that does not leak its cause",
			caller: reader(),
			mock:   expectDelete(byReader, errStoreDown),
			want:   outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}
			target := tc.target
			if target == "" {
				target = "/v1/comments/" + commentID
			}

			recorder := f.serve(tc.caller, http.MethodDelete, target, "")

			f.check(t, recorder, tc.want)
		})
	}
}
