package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

var baseTime = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

const articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"

// errStoreDown is a store failing on its own: a connection it cannot make.
var errStoreDown = errors.New("dial tcp: connection refused")

// fixedClock always answers baseTime.
type fixedClock struct{}

func (fixedClock) Now() time.Time { return baseTime }

// fixedID always answers articleID.
type fixedID struct{}

func (fixedID) NewID() string { return articleID }

// fixture is an ArticleUsecase over a mocked repository and mocked events, fresh for every case. An event
// a case does not expect fails it.
type fixture struct {
	repo    *MockArticleRepository
	events  *MockArticleEvents
	useCase *usecase.ArticleUsecase
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	controller := gomock.NewController(t)
	f := &fixture{repo: NewMockArticleRepository(controller), events: NewMockArticleEvents(controller)}
	f.useCase = usecase.NewArticleUsecase(usecase.ArticleDeps{
		Articles: f.repo,
		Events:   f.events,
		Clock:    fixedClock{},
		IDs:      fixedID{},
		PageSize: usecase.PageSize{Default: 20, Max: 100},
	})
	return f
}

// expectUpdate makes the repository update stored as Update does: apply the change, then return the result,
// or the change's error with nothing saved.
func (f *fixture) expectUpdate(stored *domain.Article) {
	f.repo.EXPECT().Update(gomock.Any(), stored.ID(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, change func(*domain.Article) error) (*domain.Article, error) {
			if err := change(stored); err != nil {
				return nil, err
			}
			return stored, nil
		})
}

// draft is the stored state of the draft Create builds from slug "hello", title "Hello" and body "x" at baseTime.
func draft() domain.Stored {
	return domain.Stored{
		ID:        articleID,
		Slug:      "hello",
		Title:     "Hello",
		Body:      "x",
		Status:    domain.StatusDraft,
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}
}

// articleFixture is the draft as the store returns it.
func articleFixture() *domain.Article {
	return domain.Restore(draft())
}

// publishedFixture is articleFixture published at baseTime.
func publishedFixture() *domain.Article {
	stored, publishedAt := draft(), baseTime
	stored.Status, stored.PublishedAt = domain.StatusPublished, &publishedAt
	return domain.Restore(stored)
}

// viewFixture is the ArticleView of articleFixture.
func viewFixture() usecase.ArticleView {
	return usecase.ArticleView{
		ID:        articleID,
		Slug:      "hello",
		Title:     "Hello",
		Body:      "x",
		Status:    domain.StatusDraft,
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}
}

// viewWith is viewFixture with one change made to it.
func viewWith(change func(*usecase.ArticleView)) usecase.ArticleView {
	view := viewFixture()
	change(&view)
	return view
}

// publishedView is the ArticleView of publishedFixture.
func publishedView() usecase.ArticleView {
	return viewWith(func(view *usecase.ArticleView) {
		publishedAt := baseTime
		view.Status, view.PublishedAt = domain.StatusPublished, &publishedAt
	})
}
