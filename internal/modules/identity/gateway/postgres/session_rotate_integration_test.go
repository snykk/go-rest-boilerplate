//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Rotate spends a live token once, even under concurrent rotations, and stores its successor in the spent
// session's family. It refuses an unknown, revoked or expired token and a token spent within the grace,
// changing nothing. A token spent before the grace revokes its family and is reported as reused, with its
// user; presented again, it is refused like any revoked token.
func TestSessionRepoRotate(t *testing.T) {
	rotateToB := func(ctx context.Context, sessions *postgres.SessionRepo) error {
		_, err := sessions.Rotate(ctx, "a", successor("b"), baseTime, grace)
		return err
	}
	revoke := func(ctx context.Context, sessions *postgres.SessionRepo) error {
		return sessions.Revoke(ctx, "a", baseTime)
	}
	tCases := []struct {
		name string
		// token is the one presented. The user holds "a", which expires at expiresAt, and "other", a session
		// of another family.
		token     string
		expiresAt time.Time
		before    func(ctx context.Context, sessions *postgres.SessionRepo) error // what happened to "a" first
		calls     int                                                             // rotations at the same time
		now       time.Time
		// wantRotated calls succeed and store their successor; the others return "" and ErrInvalidToken, or
		// the user's ID and ErrTokenReused when wantReused is set.
		wantRotated int
		wantReused  bool
		wantLive    []string // live tokens afterwards, besides the successors the calls stored
		wantRefused []string // tokens a rotation at now refuses afterwards
	}{
		{
			name:        "a live token is rotated once",
			token:       "a",
			expiresAt:   later,
			calls:       1,
			now:         baseTime,
			wantRotated: 1,
			wantLive:    []string{"other"},
			wantRefused: []string{"a"},
		},
		{
			name:        "twenty concurrent rotations of one token succeed exactly once",
			token:       "a",
			expiresAt:   later,
			calls:       20,
			now:         baseTime,
			wantRotated: 1,
			wantLive:    []string{"other"},
		},
		{
			name:      "a token spent within the grace changes nothing",
			token:     "a",
			expiresAt: later,
			before:    rotateToB,
			calls:     1,
			now:       baseTime.Add(grace / 2),
			wantLive:  []string{"b", "other"},
		},
		{
			name:        "a token spent before the grace revokes its family",
			token:       "a",
			expiresAt:   later,
			before:      rotateToB,
			calls:       1,
			wantReused:  true,
			now:         baseTime.Add(time.Minute),
			wantLive:    []string{"other"},
			wantRefused: []string{"a", "b"},
		},
		{
			name:        "a token spent before the grace and expired since revokes its family",
			token:       "a",
			expiresAt:   baseTime.Add(30 * time.Second),
			before:      rotateToB,
			calls:       1,
			wantReused:  true,
			now:         baseTime.Add(time.Minute),
			wantLive:    []string{"other"},
			wantRefused: []string{"a", "b"},
		},
		{
			name:      "an expired token is invalid",
			token:     "a",
			expiresAt: baseTime,
			calls:     1,
			now:       baseTime.Add(time.Second),
			wantLive:  []string{"other"},
		},
		{
			name:      "a revoked token is invalid",
			token:     "a",
			expiresAt: later,
			before:    revoke,
			calls:     1,
			now:       baseTime,
			wantLive:  []string{"other"},
		},
		{
			name:      "an unknown token is invalid",
			token:     "unknown",
			expiresAt: later,
			calls:     1,
			now:       baseTime,
			wantLive:  []string{"a", "other"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := storeUser(t, pool, "ada@example.com")
			family := id.New()
			storeSession(t, pool, sessionOf(user.ID(), family, "a", tc.expiresAt))
			storeSession(t, pool, sessionOf(user.ID(), id.New(), "other", later))
			sessions := postgres.NewSessionRepo(pool)
			if tc.before != nil {
				if err := tc.before(t.Context(), sessions); err != nil {
					t.Fatalf("before = %v", err)
				}
			}
			userIDs := make([]string, tc.calls)
			failures := make([]error, tc.calls)
			var running sync.WaitGroup

			for i := range tc.calls {
				running.Go(func() {
					userIDs[i], failures[i] = sessions.Rotate(
						t.Context(), tc.token, successor(nextToken(i)), tc.now, grace)
				})
			}
			running.Wait()

			var stored []string
			wantRefusal, wantRefusedID := domain.ErrInvalidToken, ""
			if tc.wantReused {
				wantRefusal, wantRefusedID = domain.ErrTokenReused, user.ID()
			}
			for i, err := range failures {
				switch {
				case err == nil && userIDs[i] == user.ID():
					stored = append(stored, nextToken(i))
				case !errors.Is(err, wantRefusal) || userIDs[i] != wantRefusedID:
					t.Fatalf("Rotate = %q, %v; want %q, nil or %q, %v",
						userIDs[i], err, user.ID(), wantRefusedID, wantRefusal)
				}
			}
			if len(stored) != tc.wantRotated {
				t.Fatalf("Rotate succeeded %d times, want %d", len(stored), tc.wantRotated)
			}
			for _, token := range stored {
				if owner, joined := ownerOf(t, pool, token); owner != user.ID() || joined != family {
					t.Errorf("successor %s: user %s, family %s; want %s, %s", token, owner, joined, user.ID(), family)
				}
			}
			wantLive := slices.Concat(stored, tc.wantLive)
			slices.Sort(wantLive)
			if got := liveTokens(t, pool, tc.now); !reflect.DeepEqual(got, wantLive) {
				t.Errorf("live sessions after Rotate = %v, want %v", got, wantLive)
			}
			for _, token := range tc.wantRefused {
				_, err := sessions.Rotate(t.Context(), token, successor("late"), tc.now, grace)
				if !errors.Is(err, domain.ErrInvalidToken) {
					t.Errorf("Rotate(%s) afterwards = %v, want %v", token, err, domain.ErrInvalidToken)
				}
			}
		})
	}
}

// nextToken is the token hash of the successor the concurrent rotation at index stores.
func nextToken(index int) string {
	return fmt.Sprintf("next-%02d", index)
}
