package rest_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// GetDraft answers 200 with a draft and 404 for an unknown article or a
// published one. An ID that is not a UUID gets 404 without the service being called.
func TestHandlerGetDraft(t *testing.T) {
	tCases := []struct {
		name       string
		path       string
		mock       func(f *fixture)
		wantStatus int
	}{
		{
			name: "a draft is returned",
			path: "/v1/articles/drafts/" + articleID,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Get(gomock.Any(), articleID, domain.StatusDraft).Return(viewFixture(), nil)
			},
			wantStatus: 200,
		},
		{
			name: "a published article is not found here",
			path: "/v1/articles/drafts/" + articleID,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Get(gomock.Any(), articleID, domain.StatusDraft).
					Return(usecase.ArticleView{}, domain.ErrNotFound)
			},
			wantStatus: 404,
		},
		{
			name:       "a malformed id never reaches the service",
			path:       "/v1/articles/drafts/not-a-uuid",
			mock:       func(*fixture) {},
			wantStatus: 404,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			recorder := f.serve("GET", tc.path, "")

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			if recorder.Code == 404 {
				if got := problemCode(t, recorder); got != "not_found" {
					t.Errorf("problem code = %q, want not_found", got)
				}
			}
		})
	}
}
