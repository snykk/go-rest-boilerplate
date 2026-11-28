package rest_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Delete answers 204 with no body, and 404 for an unknown article.
func TestHandlerDelete(t *testing.T) {
	tCases := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "a stored article is deleted", wantStatus: 204},
		{name: "an unknown article is not found", svcErr: domain.ErrNotFound, wantStatus: 404},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.useCases.EXPECT().Delete(gomock.Any(), articleID).Return(tc.svcErr)

			recorder := f.serve("DELETE", "/v1/articles/"+articleID, "")

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			if recorder.Code == 204 && recorder.Body.Len() != 0 {
				t.Errorf("body = %s, want none", recorder.Body)
			}
		})
	}
}
