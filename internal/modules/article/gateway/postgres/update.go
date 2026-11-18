package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Update applies change to the article with the given ID and saves every field while it holds the article's
// row lock, so concurrent updates of one article apply one after the other. It runs in a transaction of its
// own, or in a savepoint when the repository runs on a transaction. It returns the saved article;
// domain.ErrNotFound when there is none, including when id is not a UUID; change's error, with nothing saved;
// and domain.ErrSlugTaken when another article uses the new slug.
func (r *ArticleRepo) Update(ctx context.Context, id string,
	change func(article *domain.Article) error) (*domain.Article, error) {
	var article *domain.Article
	err := pgx.BeginFunc(ctx, r.db, func(transaction pgx.Tx) error {
		rows, err := transaction.Query(ctx, `SELECT `+articleColumns+` FROM articles WHERE id = $1 FOR UPDATE`, id)
		if err != nil {
			return translateID(err, "lock article")
		}
		row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[articleRow])
		if err != nil {
			return translateID(err, "lock article")
		}
		article = row.toDomain()

		if err := change(article); err != nil {
			return err
		}

		saved := rowOf(article)
		_, err = transaction.Exec(ctx,
			`UPDATE articles
			    SET slug = $2, title = $3, body = $4, status = $5, published_at = $6, updated_at = $7
			  WHERE id = $1`,
			saved.ID, saved.Slug, saved.Title, saved.Body, saved.Status, saved.PublishedAt, saved.UpdatedAt)
		if err != nil {
			return translate(err, "update article")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return article, nil
}
