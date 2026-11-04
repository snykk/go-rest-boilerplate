package players

import (
	"github.com/go-rest-boilerplate/internal/business/usecases/players"
)

type Handler struct {
	usecase players.Usecase
}

func NewHandler(usecase players.Usecase) Handler {
	return Handler{usecase: usecase}
}
