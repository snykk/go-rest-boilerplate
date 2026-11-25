package usecase

import (
	"context"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

// CommentView is a comment as the use cases return it: plain data, so no caller outside the use cases holds the
// entity.
type CommentView struct {
	ID, ArticleID, AuthorID string
	// AuthorName is the display name the author had when writing the comment.
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

// PageSize bounds the pages List returns.
type PageSize struct {
	// Default is the size of a page when the request asks for 0 or fewer comments.
	Default int
	// Max is the most comments a page holds.
	Max int
}

// CommentDeps are the ports and the settings a CommentUsecase works with.
type CommentDeps struct {
	Comments CommentRepository
	Articles Articles
	Authors  Authors
	Clock    Clock
	IDs      IDGenerator
	PageSize PageSize
}

// CommentUsecase runs the comment use cases. Every method that returns a CommentView or a ListOutput returns
// the zero value together with a non-nil error. It is safe for concurrent use when its ports are.
type CommentUsecase struct {
	comments CommentRepository
	articles Articles
	authors  Authors
	clock    Clock
	ids      IDGenerator
	pageSize PageSize
}

// NewCommentUsecase returns a CommentUsecase working with deps.
func NewCommentUsecase(deps CommentDeps) *CommentUsecase {
	return &CommentUsecase{
		comments: deps.Comments,
		articles: deps.Articles,
		authors:  deps.Authors,
		clock:    deps.Clock,
		ids:      deps.IDs,
		pageSize: deps.PageSize,
	}
}

// requirePublished returns domain.ErrArticleNotFound unless the article with the given ID is published, and
// the error of an Articles that cannot tell.
func (u *CommentUsecase) requirePublished(ctx context.Context, articleID string) error {
	published, err := u.articles.Published(ctx, articleID)
	if err != nil {
		return err
	}
	if !published {
		return domain.ErrArticleNotFound
	}
	return nil
}

func viewOf(comment *domain.Comment) CommentView {
	return CommentView{
		ID:         comment.ID(),
		ArticleID:  comment.ArticleID(),
		AuthorID:   comment.AuthorID(),
		AuthorName: comment.AuthorName(),
		Body:       comment.Body(),
		CreatedAt:  comment.CreatedAt(),
	}
}
