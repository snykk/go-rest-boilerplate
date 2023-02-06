package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

type UserHandler struct {
	usecase        domains.UserUsecase
	redisCache     caches.RedisCache
	ristrettoCache caches.RistrettoCache
}

func NewUserHandler(usecase domains.UserUsecase, redisCache caches.RedisCache, ristrettoCache caches.RistrettoCache) UserHandler {
	return UserHandler{
		usecase:        usecase,
		redisCache:     redisCache,
		ristrettoCache: ristrettoCache,
	}
}

func (userH UserHandler) Regis(ctx *gin.Context) {
	var UserRegisRequest requests.UserRequest
	if err := ctx.ShouldBindJSON(&UserRegisRequest); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := validators.ValidatePayloads(UserRegisRequest); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	ctxx := ctx.Request.Context()
	userDomain := UserRegisRequest.ToDomain()
	userDomainn, statusCode, err := userH.usecase.Store(ctxx, userDomain)
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	NewSuccessResponse(ctx, statusCode, "registration user success", map[string]interface{}{
		"user": responses.FromDomain(userDomainn),
	})
}

func (userH UserHandler) Login(ctx *gin.Context) {

}

func (userH UserHandler) SendOTP(ctx *gin.Context) {

}
func (userH UserHandler) VerifOTP(ctx *gin.Context) {

}
