package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

var baseTime = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

// draftWith is a draft titled "Title" with the given slug and body, created at baseTime.
func draftWith(t *testing.T, slug, body string) *domain.Article {
	t.Helper()
	article, err := domain.New("id", slug, "Title", body, baseTime)
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	return article
}

// publishedWith is draftWith published at baseTime.
func publishedWith(t *testing.T, slug, body string) *domain.Article {
	t.Helper()
	article := draftWith(t, slug, body)
	if err := article.Publish(baseTime); err != nil {
		t.Fatalf("Publish = %v", err)
	}
	return article
}

// stateOf reads every field of article through its accessors, as a gateway saving it does.
func stateOf(article *domain.Article) domain.Stored {
	state := domain.Stored{
		ID:        article.ID(),
		Slug:      article.Slug(),
		Title:     article.Title(),
		Body:      article.Body(),
		Status:    article.Status(),
		CreatedAt: article.CreatedAt(),
		UpdatedAt: article.UpdatedAt(),
	}
	if publishedAt, ok := article.PublishedAt(); ok {
		state.PublishedAt = &publishedAt
	}
	return state
}

// New builds a draft with its title trimmed, and refuses a slug, title or body outside the rules.
func TestNew(t *testing.T) {
	tCases := []struct {
		name              string
		slug, title, body string
		want              domain.Stored
		wantErr           error
	}{
		{
			name:  "a valid article is a draft with its title trimmed",
			slug:  "hello-world",
			title: "  Hello  ",
			body:  "text",
			want: domain.Stored{
				ID:        "id-1",
				Slug:      "hello-world",
				Title:     "Hello",
				Body:      "text",
				Status:    domain.StatusDraft,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
		},
		{name: "an empty slug is refused", slug: "", title: "T", wantErr: domain.ErrInvalidSlug},
		{name: "an uppercase slug is refused", slug: "Hello", title: "T", wantErr: domain.ErrInvalidSlug},
		{name: "a slug with a double hyphen is refused", slug: "a--b", title: "T", wantErr: domain.ErrInvalidSlug},
		{name: "a slug starting with a hyphen is refused", slug: "-a", title: "T", wantErr: domain.ErrInvalidSlug},
		{name: "a slug ending with a hyphen is refused", slug: "a-", title: "T", wantErr: domain.ErrInvalidSlug},
		{name: "a slug with an underscore is refused", slug: "a_b", title: "T", wantErr: domain.ErrInvalidSlug},
		{
			name:    "a slug over 100 characters is refused",
			slug:    strings.Repeat("a", 101),
			title:   "T",
			wantErr: domain.ErrInvalidSlug,
		},
		{name: "a blank title is refused", slug: "a", title: "   ", wantErr: domain.ErrInvalidTitle},
		{
			name:    "a title over 200 runes is refused",
			slug:    "a",
			title:   strings.Repeat("é", 201),
			wantErr: domain.ErrInvalidTitle,
		},
		{
			name:    "a body over 50000 runes is refused",
			slug:    "a",
			title:   "T",
			body:    strings.Repeat("é", 50001),
			wantErr: domain.ErrBodyTooLong,
		},
		{name: "a title with a NUL character is refused", slug: "a", title: "a\x00b", wantErr: domain.ErrInvalidTitle},
		{
			name:    "a body with a NUL character is refused",
			slug:    "a",
			title:   "T",
			body:    "a\x00b",
			wantErr: domain.ErrInvalidBody,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.New("id-1", tc.slug, tc.title, tc.body, baseTime)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("New = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if got != nil {
					t.Errorf("New = %+v, want no article", stateOf(got))
				}
				return
			}
			if state := stateOf(got); !reflect.DeepEqual(state, tc.want) {
				t.Errorf("New = %+v, want %+v", state, tc.want)
			}
		})
	}
}

// Restore returns the article a store kept, field for field, without checking the rules again, so an
// article stored under an older rule still loads.
func TestArticleRestore(t *testing.T) {
	later := baseTime.Add(time.Hour)
	tCases := []struct {
		name   string
		stored domain.Stored
	}{
		{
			name: "a stored draft comes back as it was stored",
			stored: domain.Stored{
				ID:        "id-1",
				Slug:      "hello",
				Title:     "Hello",
				Body:      "text",
				Status:    domain.StatusDraft,
				CreatedAt: baseTime,
				UpdatedAt: later,
			},
		},
		{
			name: "a stored published article comes back with its publication time",
			stored: domain.Stored{
				ID:          "id-2",
				Slug:        "hello",
				Title:       "Hello",
				Body:        "text",
				Status:      domain.StatusPublished,
				PublishedAt: &later,
				CreatedAt:   baseTime,
				UpdatedAt:   later,
			},
		},
		{
			name: "an article that breaks a newer rule still loads",
			stored: domain.Stored{
				ID:        "id-3",
				Slug:      "Old_Slug",
				Title:     "Hello",
				Status:    domain.StatusDraft,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.Restore(tc.stored)

			if state := stateOf(got); !reflect.DeepEqual(state, tc.stored) {
				t.Errorf("Restore = %+v, want %+v", state, tc.stored)
			}
		})
	}
}

// Publish publishes a draft that has a body, at the time given.
func TestArticlePublish(t *testing.T) {
	later := baseTime.Add(time.Hour)
	tCases := []struct {
		name    string
		article *domain.Article
		wantErr error
	}{
		{name: "a draft with a body is published", article: draftWith(t, "a", "body")},
		{
			name:    "a published article is not published again",
			article: publishedWith(t, "b", "body"),
			wantErr: domain.ErrAlreadyPublished,
		},
		{
			name:    "a draft with a blank body is not published",
			article: draftWith(t, "c", "  "),
			wantErr: domain.ErrEmptyBody,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.article.Publish(later)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Publish = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			article := tc.article
			publishedAt, published := article.PublishedAt()
			if article.Status() != domain.StatusPublished || !published || !publishedAt.Equal(later) ||
				!article.UpdatedAt().Equal(later) {
				t.Errorf("Publish left status %s, published at %v (%t), updated at %v; want published at %v",
					article.Status(), publishedAt, published, article.UpdatedAt(), later)
			}
		})
	}
}

// Edit changes only the fields given, keeps a published article's slug, and moves UpdatedAt only when
// something changed.
func TestArticleEdit(t *testing.T) {
	later := baseTime.Add(time.Minute)
	tCases := []struct {
		name              string
		article           *domain.Article
		slug, title, body *string
		wantErr           error
		wantSlug          string
		wantTitle         string
		wantUpdatedAt     time.Time
	}{
		{
			name:          "only the given title changes",
			article:       draftWith(t, "a", "body"),
			title:         new("New"),
			wantSlug:      "a",
			wantTitle:     "New",
			wantUpdatedAt: later,
		},
		{
			name:          "nothing given changes nothing, UpdatedAt included",
			article:       draftWith(t, "a", "body"),
			wantSlug:      "a",
			wantTitle:     "Title",
			wantUpdatedAt: baseTime,
		},
		{
			name:    "a published article keeps its slug",
			article: publishedWith(t, "a", "body"),
			slug:    new("renamed"),
			wantErr: domain.ErrSlugLocked,
		},
		{
			name:          "a published article's slug set to itself is not a change",
			article:       publishedWith(t, "a", "body"),
			slug:          new("a"),
			title:         new("Still editable"),
			wantSlug:      "a",
			wantTitle:     "Still editable",
			wantUpdatedAt: later,
		},
		{
			name:    "a blank title is refused",
			article: draftWith(t, "a", "body"),
			title:   new(""),
			wantErr: domain.ErrInvalidTitle,
		},
		{
			name:    "a body with a NUL character is refused",
			article: draftWith(t, "a", "body"),
			body:    new("x\x00"),
			wantErr: domain.ErrInvalidBody,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.article.Edit(tc.slug, tc.title, tc.body, later)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Edit = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			article := tc.article
			if article.Slug() != tc.wantSlug || article.Title() != tc.wantTitle || !article.UpdatedAt().Equal(tc.wantUpdatedAt) {
				t.Errorf("Edit left slug %q, title %q, updated at %v; want %q, %q, %v",
					article.Slug(), article.Title(), article.UpdatedAt(), tc.wantSlug, tc.wantTitle, tc.wantUpdatedAt)
			}
		})
	}
}
