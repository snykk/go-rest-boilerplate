//go:build integration

package postgres_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// List pages through the comments on one article, newest first, by one author when the query names one, and
// counts every match.
func TestCommentRepoList(t *testing.T) {
	repo := newRepo(t)
	articleID, ada, bob := id.New(), id.New(), id.New()
	var onArticle []*domain.Comment // oldest first: ada, bob, ada, bob
	for i := range 4 {
		author := ada
		if i%2 == 1 {
			author = bob
		}
		comment := commentOn(t, articleID, author, baseTime.Add(time.Duration(i)*time.Minute))
		store(t, repo, comment)
		onArticle = append(onArticle, comment)
	}
	store(t, repo, commentOn(t, id.New(), ada, baseTime.Add(time.Hour)))
	idsOf := func(indexes ...int) []string {
		ids := []string{}
		for _, index := range indexes {
			ids = append(ids, onArticle[index].ID())
		}
		return ids
	}
	tCases := []struct {
		name      string
		query     usecase.CommentQuery
		wantTotal int
		wantIDs   []string
	}{
		{
			name:      "the article's comments are listed newest first, without another article's",
			query:     usecase.CommentQuery{ArticleID: articleID, Limit: 10},
			wantTotal: 4,
			wantIDs:   idsOf(3, 2, 1, 0),
		},
		{
			name:      "the second page holds the next ones",
			query:     usecase.CommentQuery{ArticleID: articleID, Limit: 2, Offset: 2},
			wantTotal: 4,
			wantIDs:   idsOf(1, 0),
		},
		{
			name: "the author filter keeps one author's comments, and counts only them",
			query: usecase.CommentQuery{
				ArticleID: articleID,
				AuthorID:  ada,
				Limit:     10,
			},
			wantTotal: 2,
			wantIDs:   idsOf(2, 0),
		},
		{
			name:      "an offset past the end returns no comments and still counts them",
			query:     usecase.CommentQuery{ArticleID: articleID, Limit: 10, Offset: 10},
			wantTotal: 4,
			wantIDs:   []string{},
		},
		{
			name:      "an article with no comments has none",
			query:     usecase.CommentQuery{ArticleID: id.New(), Limit: 10},
			wantTotal: 0,
			wantIDs:   []string{},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			items, total, err := repo.List(t.Context(), tc.query)

			if err != nil {
				t.Fatalf("List = %v", err)
			}
			ids := []string{}
			for _, comment := range items {
				ids = append(ids, comment.ID())
			}
			if total != tc.wantTotal || !reflect.DeepEqual(ids, tc.wantIDs) {
				t.Errorf("List = %v (total %d), want %v (total %d)", ids, total, tc.wantIDs, tc.wantTotal)
			}
		})
	}
}
