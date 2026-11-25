package article_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/gateway/article"
)

const articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c45"

// errModuleDown is the article module failing on its own: a store it cannot reach.
var errModuleDown = errors.New("select article: dial tcp: connection refused")

// Published reports a published article as published, and a draft or an article the article module does not
// know as not published, without an error; the module's other errors pass through.
func TestArticlesPublished(t *testing.T) {
	tCases := []struct {
		name    string
		found   contract.Article
		findErr error
		want    bool
		wantErr error
	}{
		{
			name: "a published article is published",
			found: contract.Article{
				ID:        articleID,
				Slug:      "hello",
				Title:     "Hello",
				Published: true,
			},
			want: true,
		},
		{
			name:  "a draft is not published",
			found: contract.Article{ID: articleID, Slug: "hello", Title: "Hello"},
		},
		{name: "an unknown article is not published, with no error", findErr: contract.ErrNotFound},
		{name: "another error passes through", findErr: errModuleDown, wantErr: errModuleDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			api := NewMockAPI(gomock.NewController(t))
			api.EXPECT().Article(gomock.Any(), articleID).Return(tc.found, tc.findErr)

			got, err := article.NewArticles(api).Published(t.Context(), articleID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Published = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Published = %t, want %t", got, tc.want)
			}
		})
	}
}
