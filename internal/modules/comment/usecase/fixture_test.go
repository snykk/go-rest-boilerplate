package usecase_test

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

const (
	commentID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"
	articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c45"
	// authorID is the reader who wrote the comment of the fixtures.
	authorID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c46"
	// otherID is a user who did not write it.
	otherID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c47"
)

// errStoreDown is a store failing on its own: a connection it cannot make.
var errStoreDown = errors.New("dial tcp: connection refused")

// fixedClock always answers baseTime.
type fixedClock struct{}

func (fixedClock) Now() time.Time { return baseTime }

// fixedID always answers commentID.
type fixedID struct{}

func (fixedID) NewID() string { return commentID }

// fixture is a CommentUsecase over a mocked repository and mocked article and identity modules, fresh for every
// case, with pages of 20 comments by default and 100 at most. A call a case does not expect fails it.
type fixture struct {
	comments *MockCommentRepository
	articles *MockArticles
	authors  *MockAuthors
	useCase  *usecase.CommentUsecase
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	controller := gomock.NewController(t)
	f := &fixture{
		comments: NewMockCommentRepository(controller),
		articles: NewMockArticles(controller),
		authors:  NewMockAuthors(controller),
	}
	f.withPageSize(usecase.PageSize{Default: 20, Max: 100})
	return f
}

// withPageSize replaces the fixture's use case with one of the given page sizes, over the same mocks.
func (f *fixture) withPageSize(pageSize usecase.PageSize) {
	f.useCase = usecase.NewCommentUsecase(usecase.CommentDeps{
		Comments: f.comments,
		Articles: f.articles,
		Authors:  f.authors,
		Clock:    fixedClock{},
		IDs:      fixedID{},
		PageSize: pageSize,
	})
}

// expectPublished makes the article module answer that the article with the given ID is published, or not.
func (f *fixture) expectPublished(id string, published bool) {
	f.articles.EXPECT().Published(gomock.Any(), id).Return(published, nil)
}

// expectFound makes the repository find comment.
func (f *fixture) expectFound(comment *domain.Comment) {
	f.comments.EXPECT().ByID(gomock.Any(), comment.ID()).Return(comment, nil)
}

// stored is the stored state of the comment Add builds from body "Nice one." by authorID, named "Ada", on
// articleID at baseTime.
func stored() domain.Stored {
	return domain.Stored{
		ID:         commentID,
		ArticleID:  articleID,
		AuthorID:   authorID,
		AuthorName: "Ada",
		Body:       "Nice one.",
		CreatedAt:  baseTime,
	}
}

// commentFixture is the comment as the store returns it.
func commentFixture() *domain.Comment {
	return domain.Restore(stored())
}

// viewFixture is the CommentView of commentFixture.
func viewFixture() usecase.CommentView {
	return usecase.CommentView{
		ID:         commentID,
		ArticleID:  articleID,
		AuthorID:   authorID,
		AuthorName: "Ada",
		Body:       "Nice one.",
		CreatedAt:  baseTime,
	}
}
