// Package domain holds the article entity and the rules every article keeps, whatever stores or serves it.
package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Status is the publication state of an article.
type Status string

const (
	// StatusDraft is an article that is not published yet.
	StatusDraft Status = "draft"
	// StatusPublished is an article that has been published.
	StatusPublished Status = "published"
)

const (
	maxSlugLen  = 100
	maxTitleLen = 200
	maxBodyLen  = 50000
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Article is a piece of writing addressed by a unique slug. It starts as a draft and is published once. Its
// state changes only through New, Edit and Publish, so an article in memory always keeps their rules.
type Article struct {
	id          string
	slug        string
	title       string
	body        string
	status      Status
	publishedAt time.Time
	createdAt   time.Time
	updatedAt   time.Time
}

// Stored is the state of an article as a store keeps it, for Restore.
type Stored struct {
	ID, Slug, Title, Body string
	Status                Status
	// PublishedAt is nil for a draft.
	PublishedAt          *time.Time
	CreatedAt, UpdatedAt time.Time
}

// New returns a draft created at now, with its title trimmed. It returns ErrInvalidSlug, ErrInvalidTitle,
// ErrInvalidBody or ErrBodyTooLong when a field breaks its rule. No text field may contain a NUL character.
func New(id, slug, title, body string, now time.Time) (*Article, error) {
	title = strings.TrimSpace(title)
	if err := check(slug, title, body); err != nil {
		return nil, err
	}
	return &Article{
		id:        id,
		slug:      slug,
		title:     title,
		body:      body,
		status:    StatusDraft,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Restore returns the article a store kept as stored. It is for gateways that load stored articles, not a way to
// create one: it does not check the rules again, so an article stored under an older rule still loads.
func Restore(stored Stored) *Article {
	article := &Article{
		id:        stored.ID,
		slug:      stored.Slug,
		title:     stored.Title,
		body:      stored.Body,
		status:    stored.Status,
		createdAt: stored.CreatedAt,
		updatedAt: stored.UpdatedAt,
	}
	if stored.PublishedAt != nil {
		article.publishedAt = *stored.PublishedAt
	}
	return article
}

// ID returns the article's identifier.
func (a *Article) ID() string { return a.id }

// Slug returns the unique name that addresses the article.
func (a *Article) Slug() string { return a.slug }

// Title returns the article's title.
func (a *Article) Title() string { return a.title }

// Body returns the article's text.
func (a *Article) Body() string { return a.body }

// Status returns the publication state.
func (a *Article) Status() Status { return a.status }

// PublishedAt returns when the article was published, and false for a draft.
func (a *Article) PublishedAt() (time.Time, bool) {
	return a.publishedAt, a.status == StatusPublished
}

// CreatedAt returns when the article was created.
func (a *Article) CreatedAt() time.Time { return a.createdAt }

// UpdatedAt returns when the article last changed.
func (a *Article) UpdatedAt() time.Time { return a.updatedAt }

// Edit changes the fields whose pointer is not nil; a new title is trimmed. It returns ErrSlugLocked when
// the slug of a published article would change, and the errors of New when a field breaks its rule; on any
// error the article is left unchanged. Setting a field to its current value is not a change, and UpdatedAt
// becomes now only when something changed.
func (a *Article) Edit(slug, title, body *string, now time.Time) error {
	newSlug, newTitle, newBody := a.slug, a.title, a.body
	if slug != nil {
		newSlug = *slug
	}
	if title != nil {
		newTitle = strings.TrimSpace(*title)
	}
	if body != nil {
		newBody = *body
	}

	if newSlug != a.slug && a.status == StatusPublished {
		return ErrSlugLocked
	}
	if err := check(newSlug, newTitle, newBody); err != nil {
		return err
	}

	if newSlug == a.slug && newTitle == a.title && newBody == a.body {
		return nil
	}
	a.slug, a.title, a.body, a.updatedAt = newSlug, newTitle, newBody, now
	return nil
}

// Publish publishes a draft at now. It returns ErrAlreadyPublished for an article that is already published
// and ErrEmptyBody when the body is blank.
func (a *Article) Publish(now time.Time) error {
	if a.status == StatusPublished {
		return ErrAlreadyPublished
	}
	if strings.TrimSpace(a.body) == "" {
		return ErrEmptyBody
	}

	a.status, a.publishedAt, a.updatedAt = StatusPublished, now, now
	return nil
}

func check(slug, title, body string) error {
	if len(slug) > maxSlugLen || !slugPattern.MatchString(slug) {
		return ErrInvalidSlug
	}
	if title == "" || utf8.RuneCountInString(title) > maxTitleLen || strings.ContainsRune(title, 0) {
		return ErrInvalidTitle
	}
	if strings.ContainsRune(body, 0) {
		return ErrInvalidBody
	}
	if utf8.RuneCountInString(body) > maxBodyLen {
		return ErrBodyTooLong
	}
	return nil
}
