package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// ListInput selects a page of articles. An empty Status selects every status. A Limit of 0 or less means the
// default page size, a Limit over its maximum means the maximum, and a negative Offset means 0.
type ListInput struct {
	Status domain.Status
	Limit  int
	Offset int
}

// ListOutput is one page of articles, newest first, with the number of all matches and the page bounds
// that were used.
type ListOutput struct {
	Items         []ArticleView
	Total         int
	Limit, Offset int
}

// List returns the page of articles that input selects. It returns the store's error when it fails.
func (u *ArticleUsecase) List(ctx context.Context, input ListInput) (ListOutput, error) {
	query := ArticleQuery{Status: input.Status, Limit: input.Limit, Offset: max(input.Offset, 0)}
	switch {
	case query.Limit <= 0:
		query.Limit = u.pageSize.Default
	case query.Limit > u.pageSize.Max:
		query.Limit = u.pageSize.Max
	}

	items, total, err := u.articles.List(ctx, query)
	if err != nil {
		return ListOutput{}, err
	}

	var views []ArticleView
	for i := range items {
		views = append(views, viewOf(&items[i]))
	}
	return ListOutput{
		Items:  views,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	}, nil
}
