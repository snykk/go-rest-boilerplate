package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
