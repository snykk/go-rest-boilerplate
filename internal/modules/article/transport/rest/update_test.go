package rest_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Update passes only the fields in the body to the use cases, and maps their errors and a malformed body to
// problems.
func TestHandlerUpdate(t *testing.T) {
	tCases := []struct {
		name       string
		body       string
		mock       func(f *fixture)
		wantStatus int
		wantCode   string // problem code; "" on success
	}{
		{
			name: "only the title in the body is passed on",
			body: `{"title":"New"}`,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Update(gomock.Any(), usecase.UpdateInput{
					ID:    articleID,
					Title: new("New"),
				}).Return(viewFixture(), nil)
			},
			wantStatus: 200,
		},
		{
			name: "a locked slug is a conflict",
			body: `{"slug":"renamed"}`,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Update(gomock.Any(), gomock.Any()).Return(usecase.ArticleView{}, domain.ErrSlugLocked)
			},
			wantStatus: 409,
			wantCode:   "slug_locked",
		},
		{
			name:       "a malformed body is refused before the use cases are called",
			body:       `{"title":`,
			mock:       func(*fixture) {},
			wantStatus: 400,
			wantCode:   "malformed_json",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			recorder := f.serve("PATCH", "/v1/articles/"+articleID, tc.body)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			if tc.wantCode != "" && problemCode(t, recorder) != tc.wantCode {
				t.Errorf("problem code = %q, want %q", problemCode(t, recorder), tc.wantCode)
			}
		})
	}
}
