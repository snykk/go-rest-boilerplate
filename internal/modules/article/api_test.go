package article_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

const articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"

// errStoreDown is a store failing on its own: a connection it cannot make.
var errStoreDown = errors.New("dial tcp: connection refused")

// viewWith is a draft with slug "hello" and title "Hello" as the use cases return it, with one change made.
func viewWith(change func(*usecase.ArticleView)) usecase.ArticleView {
	createdAt := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	view := usecase.ArticleView{
		ID:        articleID,
		Slug:      "hello",
		Title:     "Hello",
		Body:      "x",
		Status:    domain.StatusDraft,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	change(&view)
	return view
}

// Article returns the article with the given ID, of any status, as contract.Article,
// and reports an unknown ID as contract.ErrNotFound.
func TestAPIArticle(t *testing.T) {
	tCases := []struct {
		name    string
		view    usecase.ArticleView
		getErr  error
		want    contract.Article
		wantErr error
	}{
		{
			name: "a published article is returned as published",
			view: viewWith(func(view *usecase.ArticleView) { view.Status = domain.StatusPublished }),
			want: contract.Article{
				ID:        articleID,
				Slug:      "hello",
				Title:     "Hello",
				Published: true,
			},
		},
		{
			name: "a draft is returned as not published",
			view: viewWith(func(*usecase.ArticleView) {}),
			want: contract.Article{ID: articleID, Slug: "hello", Title: "Hello"},
		},
		{name: "an unknown ID is contract.ErrNotFound", getErr: domain.ErrNotFound, wantErr: contract.ErrNotFound},
		{name: "a store failure passes through", getErr: errStoreDown, wantErr: errStoreDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewMockArticleReader(gomock.NewController(t))
			reader.EXPECT().Get(gomock.Any(), articleID, domain.Status("")).Return(tc.view, tc.getErr)

			got, err := article.NewAPI(reader).Article(t.Context(), articleID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Article = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Article = %+v, want %+v", got, tc.want)
			}
		})
	}
}
