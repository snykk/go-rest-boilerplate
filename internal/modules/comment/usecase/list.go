package usecase

import "context"

// ListInput selects a page of the comments on the article with ArticleID, only those of the user with AuthorID
// unless it is empty. A Limit of 0 or less means the default page size, a Limit over its maximum means the
// maximum, and a negative Offset means 0.
type ListInput struct {
	ArticleID string
	AuthorID  string
	Limit     int
	Offset    int
}

// ListOutput is one page of comments, newest first, with the number of all matches and the page bounds that
// were used.
type ListOutput struct {
	Items         []CommentView
	Total         int
	Limit, Offset int
}

// List returns the page of comments that input selects on a published article. It returns
// domain.ErrArticleNotFound when the article does not exist or is a draft, and the error of a port that fails.
func (u *CommentUsecase) List(ctx context.Context, input ListInput) (ListOutput, error) {
	if err := u.requirePublished(ctx, input.ArticleID); err != nil {
		return ListOutput{}, err
	}

	query := CommentQuery{
		ArticleID: input.ArticleID,
		AuthorID:  input.AuthorID,
		Limit:     input.Limit,
		Offset:    max(input.Offset, 0),
	}
	switch {
	case query.Limit <= 0:
		query.Limit = u.pageSize.Default
	case query.Limit > u.pageSize.Max:
		query.Limit = u.pageSize.Max
	}

	items, total, err := u.comments.List(ctx, query)
	if err != nil {
		return ListOutput{}, err
	}

	var views []CommentView
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
