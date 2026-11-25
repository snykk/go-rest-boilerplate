package postgres

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
)

const commentColumns = "id, article_id, author_id, author_name, body, created_at"

type commentRow struct {
	ID         string    `db:"id"`
	ArticleID  string    `db:"article_id"`
	AuthorID   string    `db:"author_id"`
	AuthorName string    `db:"author_name"`
	Body       string    `db:"body"`
	CreatedAt  time.Time `db:"created_at"`
}

func (r commentRow) toDomain() *domain.Comment {
	return domain.Restore(domain.Stored{
		ID:         r.ID,
		ArticleID:  r.ArticleID,
		AuthorID:   r.AuthorID,
		AuthorName: r.AuthorName,
		Body:       r.Body,
		CreatedAt:  r.CreatedAt.UTC(),
	})
}
