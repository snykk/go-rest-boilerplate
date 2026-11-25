//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Do keeps every write its work makes through the repositories it hands over when the work succeeds, the
// row-locked updates included, and keeps none when a step or the work fails, returning that error.
func TestUnitOfWorkDo(t *testing.T) {
	errBoom := errors.New("boom")
	register := func(ctx context.Context, repositories usecase.Repositories, user *domain.User) error {
		if err := repositories.Users.Create(ctx, user); err != nil {
			return err
		}
		if _, err := repositories.Verifications.Put(ctx, secretOf(user.ID(), "code"), 0); err != nil {
			return err
		}
		if _, err := repositories.Resets.Put(ctx, secretOf(user.ID(), "reset"), 0); err != nil {
			return err
		}
		return repositories.Sessions.Create(ctx, sessionOf(user.ID(), id.New(), "a", later))
	}
	tCases := []struct {
		name       string
		work       func(ctx context.Context, repositories usecase.Repositories, user *domain.User) error
		wantErr    error
		wantStored bool
	}{
		{
			name: "steps that all succeed are kept",
			work: func(ctx context.Context, repositories usecase.Repositories, user *domain.User) error {
				if err := register(ctx, repositories, user); err != nil {
					return err
				}
				_, err := repositories.Users.Update(ctx, user.ID(), func(loaded *domain.User) error {
					loaded.Verify(baseTime)
					return nil
				})
				return err
			},
			wantStored: true,
		},
		{
			name: "a failing step rolls back the user",
			work: func(ctx context.Context, repositories usecase.Repositories, user *domain.User) error {
				if err := register(ctx, repositories, user); err != nil {
					return err
				}
				twin, err := domain.NewUser(id.New(), user.Email(), "Twin", "bcrypt-hash", []string{"reader"}, baseTime)
				if err != nil {
					return err
				}
				return repositories.Users.Create(ctx, twin)
			},
			wantErr: domain.ErrEmailTaken,
		},
		{
			name: "work that fails rolls back every step",
			work: func(ctx context.Context, repositories usecase.Repositories, user *domain.User) error {
				if err := register(ctx, repositories, user); err != nil {
					return err
				}
				return errBoom
			},
			wantErr: errBoom,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := newPool(t)
			user := newUser(t, "ada@example.com")

			err := postgres.NewUnitOfWork(pool).Do(t.Context(), func(repositories usecase.Repositories) error {
				return tc.work(t.Context(), repositories, user)
			})

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Do = %v, want %v", err, tc.wantErr)
			}
			got, err := postgres.NewUserRepo(pool).ByID(t.Context(), user.ID())
			if stored := err == nil; stored != tc.wantStored {
				t.Fatalf("user stored = %t (%v), want %t", stored, err, tc.wantStored)
			}
			if tc.wantStored && !got.Verified() {
				t.Errorf("user verified = false, want true: the update in the unit of work was lost")
			}
			_, err = postgres.NewVerificationRepo(pool).ByHash(t.Context(), "code")
			if stored := err == nil; stored != tc.wantStored {
				t.Errorf("verification stored = %t (%v), want %t", stored, err, tc.wantStored)
			}
			_, err = postgres.NewResetRepo(pool).ByHash(t.Context(), "reset")
			if stored := err == nil; stored != tc.wantStored {
				t.Errorf("reset stored = %t (%v), want %t", stored, err, tc.wantStored)
			}
			wantLive := []string{}
			if tc.wantStored {
				wantLive = []string{"a"}
			}
			if got := liveTokens(t, pool, baseTime); !reflect.DeepEqual(got, wantLive) {
				t.Errorf("live sessions = %v, want %v", got, wantLive)
			}
		})
	}
}
