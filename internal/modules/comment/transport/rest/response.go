package rest

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

// commentResponse is one comment as the API returns it.
type commentResponse struct {
	ID        string `json:"id" example:"0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"`
	ArticleID string `json:"article_id" example:"0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c45"`
	AuthorID  string `json:"author_id" example:"0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c46"`
	// AuthorName is the display name the author had when writing the comment.
	AuthorName string    `json:"author_name" example:"Ada"`
	Body       string    `json:"body" example:"Thanks, this helped."`
	CreatedAt  time.Time `json:"created_at"`
} // @name Comment

func toResponse(view usecase.CommentView) commentResponse {
	return commentResponse{
		ID:         view.ID,
		ArticleID:  view.ArticleID,
		AuthorID:   view.AuthorID,
		AuthorName: view.AuthorName,
		Body:       view.Body,
		CreatedAt:  view.CreatedAt,
	}
}

// listResponse is one page of comments, newest first.
type listResponse struct {
	// Items is an empty array, never null, when the page holds no comments.
	Items  []commentResponse `json:"items"`
	Total  int               `json:"total" example:"1"`
	Limit  int               `json:"limit" example:"20"`
	Offset int               `json:"offset" example:"0"`
} // @name CommentList

func toListResponse(output usecase.ListOutput) listResponse {
	response := listResponse{
		Items:  make([]commentResponse, len(output.Items)),
		Total:  output.Total,
		Limit:  output.Limit,
		Offset: output.Offset,
	}
	for i, view := range output.Items {
		response.Items[i] = toResponse(view)
	}
	return response
}
