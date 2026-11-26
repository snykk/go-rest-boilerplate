// Package contract is what other modules may know of the article module: the events it publishes and the view
// its API returns. It uses the standard library only, so depending on it ties no module to this one's insides.
package contract

import (
	"errors"
	"time"
)

// ErrNotFound reports that no article has the given ID.
var ErrNotFound = errors.New("article not found")

// ArticleDeleted announces that the article with ID is gone, deleted at At.
type ArticleDeleted struct {
	ID string
	At time.Time
}

// Article is an article as other modules see it.
type Article struct {
	ID, Slug, Title string
	// Published is true once the article is published; a draft is not.
	Published bool
}
