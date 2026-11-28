package rest

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// articleResponse is one article as the API returns it.
type articleResponse struct {
	ID          string     `json:"id" example:"0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"`
	Slug        string     `json:"slug" example:"hello-world"`
	Title       string     `json:"title" example:"Hello, world"`
	Body        string     `json:"body" example:"The first article."`
	Status      string     `json:"status" enums:"draft,published" example:"draft"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
} // @name Article

func toResponse(view usecase.ArticleView) articleResponse {
	return articleResponse{
		ID:          view.ID,
		Slug:        view.Slug,
		Title:       view.Title,
		Body:        view.Body,
		Status:      string(view.Status),
		PublishedAt: view.PublishedAt,
		CreatedAt:   view.CreatedAt,
		UpdatedAt:   view.UpdatedAt,
	}
}

// listResponse is one page of articles, newest first.
type listResponse struct {
	// Items is an empty array, never null, when the page holds no articles.
	Items  []articleResponse `json:"items"`
	Total  int               `json:"total" example:"1"`
	Limit  int               `json:"limit" example:"20"`
	Offset int               `json:"offset" example:"0"`
} // @name ArticleList

func toListResponse(output usecase.ListOutput) listResponse {
	response := listResponse{
		Items:  make([]articleResponse, len(output.Items)),
		Total:  output.Total,
		Limit:  output.Limit,
		Offset: output.Offset,
	}
	for i, view := range output.Items {
		response.Items[i] = toResponse(view)
	}
	return response
}
