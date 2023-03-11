package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"staus":   true,
		"message": "v1 online...",
	})
}
