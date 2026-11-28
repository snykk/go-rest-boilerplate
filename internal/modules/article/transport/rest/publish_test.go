package rest_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Publish answers 200 with the published article, and maps a refused publication to its problem.
func TestHandlerPublish(t *testing.T) {
	tCases := []struct {
		name       string
		svcErr     error
		wantStatus int
		wantCode   string
	}{
		{name: "a draft is published", wantStatus: 200},
		{
			name:       "a published article is a conflict",
			svcErr:     domain.ErrAlreadyPublished,
			wantStatus: 409,
			wantCode:   "already_published",
		},
		{
			name:       "a draft without a body is unprocessable",
			svcErr:     domain.ErrEmptyBody,
			wantStatus: 422,
			wantCode:   "empty_body",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			var published usecase.ArticleView
			if tc.svcErr == nil {
				published = viewFixture()
			}
			f.useCases.EXPECT().Publish(gomock.Any(), articleID).Return(published, tc.svcErr)

			recorder := f.serve("POST", "/v1/articles/"+articleID+"/publish", "")

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			if tc.wantCode != "" {
				if got := problemCode(t, recorder); got != tc.wantCode {
					t.Errorf("problem code = %q, want %q", got, tc.wantCode)
				}
			}
		})
	}
}
