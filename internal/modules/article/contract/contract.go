// Package contract is what other modules may know of the article module: the view its API returns and the
// errors it reports. It uses the standard library only, so depending on it ties no module to this one's insides.
package contract

import "errors"

// ErrNotFound reports that no article has the given ID.
var ErrNotFound = errors.New("article not found")

// Article is an article as other modules see it.
type Article struct {
	ID, Slug, Title string
	// Published is true once the article is published; a draft is not.
	Published bool
}
