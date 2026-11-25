package event_test

import (
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	articlecontract "github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
)

// Subscribe makes the bus run DeleteForArticle with the article's ID when ArticleDeleted is published, and has
// the bus log the use case's failure, since a handler has no caller to return it to.
func TestSubscribe(t *testing.T) {
	tCases := []struct {
		name     string
		mock     func(f *fixture)
		wantLogs []string
	}{
		{
			name: "a deleted article's comments are deleted",
			mock: func(f *fixture) {
				f.useCases.EXPECT().DeleteForArticle(gomock.Any(), articleID).Return(nil)
			},
		},
		{
			name: "a failure is returned to the bus, which logs it",
			mock: func(f *fixture) {
				f.useCases.EXPECT().DeleteForArticle(gomock.Any(), articleID).Return(errStoreDown)
			},
			wantLogs: []string{"comment.article-deleted", articleID, errStoreDown.Error()},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			f.bus.Publish(t.Context(), articlecontract.ArticleDeleted{
				ID: articleID,
				At: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC),
			})

			for _, want := range tc.wantLogs {
				if !strings.Contains(f.logs.String(), want) {
					t.Errorf("logs = %s, want %q logged", f.logs, want)
				}
			}
			if len(tc.wantLogs) == 0 && f.logs.Len() != 0 {
				t.Errorf("logs = %s, want none", f.logs)
			}
		})
	}
}
