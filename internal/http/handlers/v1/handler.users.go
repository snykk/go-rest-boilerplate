package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

type UserHandler struct {
	usecase V1Domains.UserUsecase
}

func NewUserHandler(usecase V1Domains.UserUsecase) UserHandler {
	return UserHandler{
		usecase: usecase,
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
		RespondWithError(ctx, err)
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
		RespondWithError(ctx, err)
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

	err := userH.usecase.SendOTP(ctx.Request.Context(), userOTP.Email)
	if err != nil {
		RespondWithError(ctx, err)
		return
	}

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

	err := userH.usecase.VerifyOTP(ctx.Request.Context(), userOTP.Email, userOTP.Code)
	if err != nil {
		RespondWithError(ctx, err)
		return
	}

	NewSuccessResponse(ctx, http.StatusOK, "otp verification success", nil)
}

func (userH UserHandler) Refresh(ctx *gin.Context) {
	var req requests.UserRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userDomain, err := userH.usecase.Refresh(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		RespondWithError(ctx, err)
		return
	}

	NewSuccessResponse(ctx, http.StatusOK, "token refreshed", responses.FromV1Domain(userDomain))
}

func (userH UserHandler) Logout(ctx *gin.Context) {
	var req requests.UserRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := userH.usecase.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		RespondWithError(ctx, err)
		return
	}

	NewSuccessResponse(ctx, http.StatusOK, "logout success", nil)
}

func (userH UserHandler) GetUserData(ctx *gin.Context) {
	user, err := auth.CurrentUserFromContext(ctx)
	if err != nil {
		NewErrorResponse(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	userDom, err := userH.usecase.GetByEmail(ctx.Request.Context(), user.Email)
	if err != nil {
		RespondWithError(ctx, err)
		return
	}

	NewSuccessResponse(ctx, http.StatusOK, "user data fetched successfully", map[string]interface{}{
		"user": responses.FromV1Domain(userDom),
	})
}
