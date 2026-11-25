// Package domain holds the comment entity and the rules every comment keeps, whatever stores or serves it.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const maxBodyLen = 2000

// Comment is what a user wrote on an article, with the display name the author had when writing it. A comment
// does not change once written: New creates it and checks its rules, Restore loads a stored one, and no method
// changes it.
type Comment struct {
	id         string
	articleID  string
	authorID   string
	authorName string
	body       string
	createdAt  time.Time
}

// Stored is the state of a comment as a store keeps it, for Restore.
type Stored struct {
	ID, ArticleID, AuthorID, AuthorName, Body string
	CreatedAt                                 time.Time
}

// New returns a comment created at now on the article with articleID by the user with authorID, shown as
// authorName, with its body trimmed. It returns ErrInvalidBody unless the trimmed body is 1 to 2000 characters,
// none of them NUL.
func New(id, articleID, authorID, authorName, body string, now time.Time) (*Comment, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > maxBodyLen || strings.ContainsRune(body, 0) {
		return nil, ErrInvalidBody
	}
	return &Comment{
		id:         id,
		articleID:  articleID,
		authorID:   authorID,
		authorName: authorName,
		body:       body,
		createdAt:  now,
	}, nil
}

// Restore returns the comment a store kept as stored. It is for gateways that load stored comments, not a way to
// create one: it does not check the rules again, so a comment stored under an older rule still loads.
func Restore(stored Stored) *Comment {
	return &Comment{
		id:         stored.ID,
		articleID:  stored.ArticleID,
		authorID:   stored.AuthorID,
		authorName: stored.AuthorName,
		body:       stored.Body,
		createdAt:  stored.CreatedAt,
	}
}

// ID returns the comment's identifier.
func (c *Comment) ID() string { return c.id }

// ArticleID returns the identifier of the article the comment is on.
func (c *Comment) ArticleID() string { return c.articleID }

// AuthorID returns the identifier of the user who wrote the comment.
func (c *Comment) AuthorID() string { return c.authorID }

// AuthorName returns the display name the author had when writing the comment.
func (c *Comment) AuthorName() string { return c.authorName }

// Body returns the comment's text.
func (c *Comment) Body() string { return c.body }

// CreatedAt returns when the comment was written.
func (c *Comment) CreatedAt() time.Time { return c.createdAt }
