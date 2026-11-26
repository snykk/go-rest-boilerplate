package domain

import "errors"

// Errors of the article rules and of the article store. Their messages are written for API clients and are
// safe to return to them.
var (
	ErrNotFound         = errors.New("article not found")
	ErrSlugTaken        = errors.New("slug is already used by another article")
	ErrInvalidSlug      = errors.New("slug must be 1-100 lowercase letters, digits or single hyphens")
	ErrInvalidTitle     = errors.New("title must be 1-200 characters, none of them NUL")
	ErrInvalidBody      = errors.New("body must not contain NUL characters")
	ErrBodyTooLong      = errors.New("body must be at most 50000 characters")
	ErrEmptyBody        = errors.New("an article needs a body to be published")
	ErrAlreadyPublished = errors.New("article is already published")
	ErrSlugLocked       = errors.New("the slug of a published article cannot change")
)
