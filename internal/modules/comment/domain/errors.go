package domain

import "errors"

// Errors of the comment rules, of the comment store and of the modules a comment refers to. Their messages are
// written for API clients and are safe to return to them.
var (
	ErrNotFound = errors.New("comment not found")
	// ErrArticleNotFound reports that no published article has the given ID: a draft is not found either.
	ErrArticleNotFound = errors.New("article not found")
	// ErrAuthorNotFound reports that no user has the ID a comment's author signed in with.
	ErrAuthorNotFound = errors.New("author not found")
	// ErrForbidden reports a caller who may not delete a comment: only its author and editors may.
	ErrForbidden   = errors.New("only the author of a comment or an editor may delete it")
	ErrInvalidBody = errors.New("body must be 1-2000 characters, none of them NUL")
)
