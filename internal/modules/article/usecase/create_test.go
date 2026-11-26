package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Create stores a new article as a draft, refusing an invalid slug before the store is asked and a slug
// another article uses.
func TestCreate(t *testing.T) {
	tCases := []struct {
		name    string
		in      usecase.CreateInput
		mock    func(f *fixture)
		want    usecase.ArticleView
		wantErr error
	}{
		{
			name: "a new article is stored as a draft with its title trimmed",
			in: usecase.CreateInput{
				Slug:  "hello",
				Title: " Hello ",
				Body:  "x",
			},
			mock: func(f *fixture) { f.repo.EXPECT().Create(gomock.Any(), articleFixture()).Return(nil) },
			want: viewFixture(),
		},
		{
			name: "a slug another article uses is refused",
			in: usecase.CreateInput{
				Slug:  "hello",
				Title: "Hello",
			},
			mock:    func(f *fixture) { f.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrSlugTaken) },
			wantErr: domain.ErrSlugTaken,
		},
		{
			name: "an invalid slug is refused before the store is asked",
			in: usecase.CreateInput{
				Slug:  "Bad Slug",
				Title: "Hello",
			},
			mock:    func(*fixture) {},
			wantErr: domain.ErrInvalidSlug,
		},
		{
			name: "a store that cannot be reached fails the call",
			in: usecase.CreateInput{
				Slug:  "hello",
				Title: "Hello",
			},
			mock:    func(f *fixture) { f.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errStoreDown) },
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Create(t.Context(), tc.in)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Create = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Create = %+v, want %+v", got, tc.want)
			}
		})
	}
}
