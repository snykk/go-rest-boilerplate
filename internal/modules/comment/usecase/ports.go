// Package usecase runs the comment use cases and declares the ports they need from the outside world.
package usecase

import (
	"context"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

//go:generate go tool mockgen -source=ports.go -destination=ports_mock_test.go -package=usecase_test

// CommentRepository stores and loads comments.
type CommentRepository interface {
	// Create stores a new comment.
	Create(ctx context.Context, comment *domain.Comment) error
	// ByID returns the comment with the given ID, or domain.ErrNotFound.
	ByID(ctx context.Context, id string) (*domain.Comment, error)
	// Delete removes the comment with the given ID. It returns domain.ErrNotFound for an unknown ID.
	Delete(ctx context.Context, id string) error
	// DeleteForArticle removes every comment on the article with the given ID. An article with no comments is
	// not an error, so calling it again changes nothing.
	DeleteForArticle(ctx context.Context, articleID string) error
	// List returns one page of the comments matching query, newest first, and the number of all matches.
	List(ctx context.Context, query CommentQuery) (items []domain.Comment, total int, err error)
}

// Articles tells whether an article takes comments.
type Articles interface {
	// Published reports whether the article with the given ID exists and is published. It returns false, and
	// no error, for an unknown ID and for a draft, and an error only when it cannot tell.
	Published(ctx context.Context, id string) (bool, error)
}

// Authors tells who the authors of comments are.
type Authors interface {
	// DisplayName returns the name the user with the given ID shows to others. It returns
	// domain.ErrAuthorNotFound for an unknown ID, and another error when it cannot tell.
	DisplayName(ctx context.Context, userID string) (string, error)
}

// Clock provides the current time.
type Clock interface {
	// Now returns the current time in UTC, at the precision the comment store keeps, so a stored time
	// reads back equal.
	Now() time.Time
}

// IDGenerator provides identifiers for new comments.
type IDGenerator interface {
	// NewID returns an identifier no other call returns, in the form CommentRepository accepts as an ID.
	NewID() string
}

// CommentQuery narrows CommentRepository.List to the comments on the article with ArticleID, and to those of
// the user with AuthorID unless it is empty.
type CommentQuery struct {
	ArticleID string
	AuthorID  string
	Limit     int
	Offset    int
}
