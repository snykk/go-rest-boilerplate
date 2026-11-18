package usecase

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// ArticleView is an article as the use cases return it: plain data, so no caller outside the use cases holds
// the entity or changes it.
type ArticleView struct {
	ID, Slug, Title, Body string
	Status                domain.Status
	// PublishedAt is nil for a draft.
	PublishedAt          *time.Time
	CreatedAt, UpdatedAt time.Time
}

// PageSize bounds the pages List returns.
type PageSize struct {
	// Default is the size of a page when the request asks for 0 or fewer articles.
	Default int
	// Max is the most articles a page holds.
	Max int
}

// ArticleDeps are the ports and the settings an ArticleUsecase works with.
type ArticleDeps struct {
	Articles ArticleRepository
	Clock    Clock
	IDs      IDGenerator
	PageSize PageSize
}

// ArticleUsecase runs the article use cases. Every method that returns an article returns the zero
// ArticleView together with a non-nil error.
type ArticleUsecase struct {
	articles ArticleRepository
	clock    Clock
	ids      IDGenerator
	pageSize PageSize
}

// NewArticleUsecase returns an ArticleUsecase working with deps.
func NewArticleUsecase(deps ArticleDeps) *ArticleUsecase {
	return &ArticleUsecase{
		articles: deps.Articles,
		clock:    deps.Clock,
		ids:      deps.IDs,
		pageSize: deps.PageSize,
	}
}

func viewOf(article *domain.Article) ArticleView {
	view := ArticleView{
		ID:        article.ID(),
		Slug:      article.Slug(),
		Title:     article.Title(),
		Body:      article.Body(),
		Status:    article.Status(),
		CreatedAt: article.CreatedAt(),
		UpdatedAt: article.UpdatedAt(),
	}
	if publishedAt, ok := article.PublishedAt(); ok {
		view.PublishedAt = &publishedAt
	}
	return view
}
