package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

type UserHandler struct {
	usecase        V1Domains.UserUsecase
	redisCache     caches.RedisCache
	ristrettoCache caches.RistrettoCache
}

func NewUserHandler(usecase V1Domains.UserUsecase, redisCache caches.RedisCache, ristrettoCache caches.RistrettoCache) UserHandler {
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

	userDomain := UserRegisRequest.ToV1Domain()
	userDomainn, statusCode, err := userH.usecase.Store(ctx.Request.Context(), userDomain)
	fmt.Println(userDomain, statusCode, err)
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	NewSuccessResponse(ctx, statusCode, "registration user success", map[string]interface{}{
		"user": responses.FromV1Domain(userDomainn),
	})
}

func (userH UserHandler) Login(ctx *gin.Context) {
	var UserLoginRequest requests.UserLoginRequest
	if err := ctx.ShouldBindJSON(&UserLoginRequest); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := validators.ValidatePayloads(UserLoginRequest); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userDomain, statusCode, err := userH.usecase.Login(ctx.Request.Context(), UserLoginRequest.ToV1Domain())
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	NewSuccessResponse(ctx, statusCode, "login success", responses.FromV1Domain(userDomain))
}

func (userH UserHandler) SendOTP(ctx *gin.Context) {
	var userOTP requests.UserSendOTPRequest

	if err := ctx.ShouldBindJSON(&userOTP); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := validators.ValidatePayloads(userOTP); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	otpCode, statusCode, err := userH.usecase.SendOTP(ctx.Request.Context(), userOTP.Email)
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	otpKey := fmt.Sprintf("user_otp:%s", userOTP.Email)
	go userH.redisCache.Set(otpKey, otpCode)

	NewSuccessResponse(ctx, statusCode, fmt.Sprintf("otp code has been send to %s", userOTP.Email), nil)
}

func (userH UserHandler) VerifOTP(ctx *gin.Context) {
	var userOTP requests.UserVerifOTPRequest

	if err := ctx.ShouldBindJSON(&userOTP); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := validators.ValidatePayloads(userOTP); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	otpKey := fmt.Sprintf("user_otp:%s", userOTP.Email)
	otpRedis, err := userH.redisCache.Get(otpKey)
	if err != nil {
		NewErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	statusCode, err := userH.usecase.VerifOTP(ctx.Request.Context(), userOTP.Email, userOTP.Code, otpRedis)
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	statusCode, err = userH.usecase.ActivateUser(ctx.Request.Context(), userOTP.Email)
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	go userH.redisCache.Del(otpKey)
	go userH.ristrettoCache.Del("users")

	NewSuccessResponse(ctx, statusCode, "otp verification success", nil)
}

func (c UserHandler) GetUserData(ctx *gin.Context) {
	// get authenticated user from context
	userClaims := ctx.MustGet(constants.CtxAuthenticatedUserKey).(jwt.JwtCustomClaim)
	if val := c.ristrettoCache.Get(fmt.Sprintf("user/%s", userClaims.Email)); val != nil {
		NewSuccessResponse(ctx, http.StatusOK, "user data fetched successfully", map[string]interface{}{
			"user": val,
		})
		return
	}

	ctxx := ctx.Request.Context()
	userDom, statusCode, err := c.usecase.GetByEmail(ctxx, userClaims.Email)
	if err != nil {
		NewErrorResponse(ctx, statusCode, err.Error())
		return
	}

	userResponse := responses.FromV1Domain(userDom)

	go c.ristrettoCache.Set(fmt.Sprintf("user/%s", userClaims.Email), userResponse)

	NewSuccessResponse(ctx, statusCode, "user data fetched successfully", map[string]interface{}{
		"user": userResponse,
	})

}
