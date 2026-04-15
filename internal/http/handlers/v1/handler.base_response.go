package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
)

type BaseResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewSuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, BaseResponse{
		Status:  true,
		Message: message,
		Data:    data,
	})
}

func NewErrorResponse(c *gin.Context, statusCode int, err string) {
	c.JSON(statusCode, BaseResponse{
		Status:  false,
		Message: err,
	})

}

func NewAbortResponse(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": false, "message": message})
}

// mapDomainErrorToHTTP converts a domain error to an HTTP status code.
func mapDomainErrorToHTTP(err error) int {
	var domErr *constants.DomainError
	if errors.As(err, &domErr) {
		switch domErr.Type {
		case constants.ErrTypeNotFound:
			return http.StatusNotFound
		case constants.ErrTypeUnauthorized:
			return http.StatusUnauthorized
		case constants.ErrTypeForbidden:
			return http.StatusForbidden
		case constants.ErrTypeConflict:
			return http.StatusConflict
		case constants.ErrTypeBadRequest:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}
