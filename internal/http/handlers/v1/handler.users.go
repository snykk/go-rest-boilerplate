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

func (userH UserHandler) Register(ctx *gin.Context) {
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
	userDomainn, err := userH.usecase.Store(ctx.Request.Context(), userDomain)
	if err != nil {
		NewErrorResponse(ctx, mapDomainErrorToHTTP(err), err.Error())
		return
	}

	NewSuccessResponse(ctx, http.StatusCreated, "registration user success", map[string]interface{}{
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

	userDomain, err := userH.usecase.Login(ctx.Request.Context(), UserLoginRequest.ToV1Domain())
	if err != nil {
		NewErrorResponse(ctx, mapDomainErrorToHTTP(err), err.Error())
		return
	}

	NewSuccessResponse(ctx, http.StatusOK, "login success", responses.FromV1Domain(userDomain))
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

	otpCode, err := userH.usecase.SendOTP(ctx.Request.Context(), userOTP.Email)
	if err != nil {
		NewErrorResponse(ctx, mapDomainErrorToHTTP(err), err.Error())
		return
	}

	otpKey := fmt.Sprintf("user_otp:%s", userOTP.Email)
	go userH.redisCache.Set(otpKey, otpCode)

	NewSuccessResponse(ctx, http.StatusOK, fmt.Sprintf("otp code has been send to %s", userOTP.Email), nil)
}

func (userH UserHandler) VerifyOTP(ctx *gin.Context) {
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

	err = userH.usecase.VerifOTP(ctx.Request.Context(), userOTP.Email, userOTP.Code, otpRedis)
	if err != nil {
		NewErrorResponse(ctx, mapDomainErrorToHTTP(err), err.Error())
		return
	}

	err = userH.usecase.ActivateUser(ctx.Request.Context(), userOTP.Email)
	if err != nil {
		NewErrorResponse(ctx, mapDomainErrorToHTTP(err), err.Error())
		return
	}

	go userH.redisCache.Del(otpKey)
	go userH.ristrettoCache.Del("users")

	NewSuccessResponse(ctx, http.StatusOK, "otp verification success", nil)
}

func (userH UserHandler) GetUserData(ctx *gin.Context) {
	claimsVal, exists := ctx.Get(constants.CtxAuthenticatedUserKey)
	if !exists {
		NewErrorResponse(ctx, http.StatusUnauthorized, "user not authenticated")
		return
	}
	userClaims, ok := claimsVal.(jwt.JwtCustomClaim)
	if !ok {
		NewErrorResponse(ctx, http.StatusInternalServerError, "invalid user claims")
		return
	}

	if val := userH.ristrettoCache.Get(fmt.Sprintf("user/%s", userClaims.Email)); val != nil {
		NewSuccessResponse(ctx, http.StatusOK, "user data fetched successfully", map[string]interface{}{
			"user": val,
		})
		return
	}

	ctxx := ctx.Request.Context()
	userDom, err := userH.usecase.GetByEmail(ctxx, userClaims.Email)
	if err != nil {
		NewErrorResponse(ctx, mapDomainErrorToHTTP(err), err.Error())
		return
	}

	userResponse := responses.FromV1Domain(userDom)

	go userH.ristrettoCache.Set(fmt.Sprintf("user/%s", userClaims.Email), userResponse)

	NewSuccessResponse(ctx, http.StatusOK, "user data fetched successfully", map[string]interface{}{
		"user": userResponse,
	})
}
