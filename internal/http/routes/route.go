package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RootHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":  true,
		"message": "welcome to an amazing api",
	})
}
