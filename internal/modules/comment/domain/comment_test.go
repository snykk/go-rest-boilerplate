package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

// stateOf reads every field of comment through its accessors, as a gateway saving it does.
func stateOf(comment *domain.Comment) domain.Stored {
	return domain.Stored{
		ID:         comment.ID(),
		ArticleID:  comment.ArticleID(),
		AuthorID:   comment.AuthorID(),
		AuthorName: comment.AuthorName(),
		Body:       comment.Body(),
		CreatedAt:  comment.CreatedAt(),
	}
}

// New builds a comment with its body trimmed, and refuses a body that is blank, longer than 2000 characters or
// holds a NUL character.
func TestNew(t *testing.T) {
	tCases := []struct {
		name    string
		body    string
		want    domain.Stored
		wantErr error
	}{
		{
			name: "a valid comment keeps its article, its author and its trimmed body",
			body: "  Nice one.\n",
			want: domain.Stored{
				ID:         "c1",
				ArticleID:  "a1",
				AuthorID:   "u1",
				AuthorName: "Ada",
				Body:       "Nice one.",
				CreatedAt:  baseTime,
			},
		},
		{
			name: "a body of 2000 characters is accepted",
			body: strings.Repeat("é", 2000),
			want: domain.Stored{
				ID:         "c1",
				ArticleID:  "a1",
				AuthorID:   "u1",
				AuthorName: "Ada",
				Body:       strings.Repeat("é", 2000),
				CreatedAt:  baseTime,
			},
		},
		{name: "an empty body is refused", body: "", wantErr: domain.ErrInvalidBody},
		{name: "a blank body is refused", body: " \n\t ", wantErr: domain.ErrInvalidBody},
		{name: "a body over 2000 characters is refused", body: strings.Repeat("é", 2001), wantErr: domain.ErrInvalidBody},
		{name: "a body with a NUL character is refused", body: "a\x00b", wantErr: domain.ErrInvalidBody},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.New("c1", "a1", "u1", "Ada", tc.body, baseTime)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("New = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if got != nil {
					t.Errorf("New = %+v, want no comment", stateOf(got))
				}
				return
			}
			if state := stateOf(got); !reflect.DeepEqual(state, tc.want) {
				t.Errorf("New = %+v, want %+v", state, tc.want)
			}
		})
	}
}

// Restore returns the comment a store kept, field for field, without checking the rules again, so a comment
// stored under an older rule still loads.
func TestCommentRestore(t *testing.T) {
	tCases := []struct {
		name   string
		stored domain.Stored
	}{
		{
			name: "a stored comment comes back as it was stored",
			stored: domain.Stored{
				ID:         "c1",
				ArticleID:  "a1",
				AuthorID:   "u1",
				AuthorName: "Ada",
				Body:       "Nice one.",
				CreatedAt:  baseTime,
			},
		},
		{
			name: "a comment that breaks a newer rule still loads",
			stored: domain.Stored{
				ID:         "c2",
				ArticleID:  "a1",
				AuthorID:   "u1",
				AuthorName: "Ada",
				Body:       strings.Repeat("x", 5000),
				CreatedAt:  baseTime,
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
