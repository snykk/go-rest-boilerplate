package v1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/responses"
	"github.com/snykk/go-rest-boilerplate/pkg/audit"
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

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user account in inactive state. The caller must follow up with /auth/send-otp + /auth/verify-otp to activate the account before /auth/login will succeed.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserRequest  true  "Registration payload"
// @Success      201      {object}  v1.BaseResponse{data=responses.UserResponse}  "User created"
// @Failure      400      {object}  v1.BaseResponse                                "Malformed JSON body"
// @Failure      409      {object}  v1.BaseResponse                                "Email or username already in use"
// @Failure      422      {object}  v1.BaseResponse                                "Validation error (per-field detail in data.errors)"
// @Failure      500      {object}  v1.BaseResponse                                "Internal error"
// @Router       /auth/register [post]
func (userH UserHandler) Register(ctx *gin.Context) {
	var UserRegisRequest requests.UserRequest
	if err := ctx.ShouldBindJSON(&UserRegisRequest); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validators.ValidatePayloads(UserRegisRequest); err != nil {
		RespondWithError(ctx, err)
		return
	}

	userDomain := UserRegisRequest.ToV1Domain()
	userDomainn, err := userH.usecase.Store(ctx.Request.Context(), userDomain)
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventRegister
		ev.Email = UserRegisRequest.Email
		ev.Reason = err.Error()
		audit.Record(ev)
		RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventRegister
	ev.Success = true
	ev.UserID = userDomainn.ID
	ev.Email = userDomainn.Email
	audit.Record(ev)

	NewSuccessResponse(ctx, http.StatusCreated, "registration user success", map[string]interface{}{
		"user": responses.FromV1Domain(userDomainn),
	})
}

// Login godoc
// @Summary      Authenticate and issue a token pair
// @Description  Returns an access token (short TTL) and a refresh token (longer TTL). Wrong password and unknown email take the same wall time to prevent user enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserLoginRequest  true  "Login credentials"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "Tokens issued"
// @Failure      400      {object}  v1.BaseResponse                                "Malformed JSON body"
// @Failure      401      {object}  v1.BaseResponse                                "Invalid email or password"
// @Failure      403      {object}  v1.BaseResponse                                "Account not yet activated"
// @Failure      422      {object}  v1.BaseResponse                                "Validation error"
// @Router       /auth/login [post]
func (userH UserHandler) Login(ctx *gin.Context) {
	var UserLoginRequest requests.UserLoginRequest
	if err := ctx.ShouldBindJSON(&UserLoginRequest); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validators.ValidatePayloads(UserLoginRequest); err != nil {
		RespondWithError(ctx, err)
		return
	}

	userDomain, err := userH.usecase.Login(ctx.Request.Context(), UserLoginRequest.ToV1Domain())
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventLoginFailure
		ev.Email = UserLoginRequest.Email
		ev.Reason = err.Error()
		audit.Record(ev)
		RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventLoginSuccess
	ev.Success = true
	ev.UserID = userDomain.ID
	ev.Email = userDomain.Email
	audit.Record(ev)

	NewSuccessResponse(ctx, http.StatusOK, "login success", responses.FromV1Domain(userDomain))
}

// SendOTP godoc
// @Summary      Send an OTP code to the user's email
// @Description  Generates a 6-digit OTP, stores it in Redis with a TTL, and enqueues the email via the async mailer. The HTTP response returns on enqueue, not on actual SMTP delivery.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserSendOTPRequest  true  "Email to send OTP to"
// @Success      200      {object}  v1.BaseResponse  "OTP enqueued"
// @Failure      404      {object}  v1.BaseResponse  "Email not registered"
// @Failure      400      {object}  v1.BaseResponse  "Account already activated"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Failure      500      {object}  v1.BaseResponse  "Failed to enqueue mail"
// @Router       /auth/send-otp [post]
func (userH UserHandler) SendOTP(ctx *gin.Context) {
	var userOTP requests.UserSendOTPRequest

	if err := ctx.ShouldBindJSON(&userOTP); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validators.ValidatePayloads(userOTP); err != nil {
		RespondWithError(ctx, err)
		return
	}

	err := userH.usecase.SendOTP(ctx.Request.Context(), userOTP.Email)
	if err != nil {
		RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventOTPSent
	ev.Success = true
	ev.Email = userOTP.Email
	audit.Record(ev)

	NewSuccessResponse(ctx, http.StatusOK, fmt.Sprintf("otp code has been send to %s", userOTP.Email), nil)
}

// VerifyOTP godoc
// @Summary      Verify an OTP code and activate the account
// @Description  Validates the supplied code against Redis and flips the user's active flag to true on success. Brute-force-guarded — after OTP_MAX_ATTEMPTS failures (default 5) the email is locked out for the OTP TTL window even with the correct code.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserVerifOTPRequest  true  "Email + OTP code"
// @Success      200      {object}  v1.BaseResponse  "Account activated"
// @Failure      400      {object}  v1.BaseResponse  "Invalid OTP code"
// @Failure      403      {object}  v1.BaseResponse  "Locked out — too many invalid attempts"
// @Failure      404      {object}  v1.BaseResponse  "Email not registered"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/verify-otp [post]
func (userH UserHandler) VerifyOTP(ctx *gin.Context) {
	var userOTP requests.UserVerifOTPRequest

	if err := ctx.ShouldBindJSON(&userOTP); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validators.ValidatePayloads(userOTP); err != nil {
		RespondWithError(ctx, err)
		return
	}

	err := userH.usecase.VerifyOTP(ctx.Request.Context(), userOTP.Email, userOTP.Code)
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Email = userOTP.Email
		ev.Reason = err.Error()
		// Distinguish "you got it wrong" from "we locked the account"
		// — rate-limit + alerting want different signals on each.
		var domErr *apperror.DomainError
		if errors.As(err, &domErr) && domErr.Type == apperror.ErrTypeForbidden {
			ev.Type = audit.EventOTPLockout
		} else {
			ev.Type = audit.EventOTPVerifyFail
		}
		audit.Record(ev)
		RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventOTPVerifyOK
	ev.Success = true
	ev.Email = userOTP.Email
	audit.Record(ev)

	NewSuccessResponse(ctx, http.StatusOK, "otp verification success", nil)
}

// Refresh godoc
// @Summary      Rotate the refresh token, return a new pair
// @Description  Verifies the supplied refresh token, mints a new access+refresh pair, and revokes the old jti in Redis. Replaying an already-rotated token returns 401.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserRefreshRequest  true  "Refresh token"
// @Success      200      {object}  v1.BaseResponse{data=responses.UserResponse}  "New token pair"
// @Failure      401      {object}  v1.BaseResponse  "Refresh token invalid, expired, or already revoked"
// @Failure      403      {object}  v1.BaseResponse  "Account no longer active"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/refresh [post]
func (userH UserHandler) Refresh(ctx *gin.Context) {
	var req requests.UserRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		RespondWithError(ctx, err)
		return
	}

	userDomain, err := userH.usecase.Refresh(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		ev := auditFromGin(ctx)
		ev.Type = audit.EventRefreshFail
		ev.Reason = err.Error()
		audit.Record(ev)
		RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventRefreshOK
	ev.Success = true
	ev.UserID = userDomain.ID
	ev.Email = userDomain.Email
	audit.Record(ev)

	NewSuccessResponse(ctx, http.StatusOK, "token refreshed", responses.FromV1Domain(userDomain))
}

// Logout godoc
// @Summary      Revoke the refresh token
// @Description  Deletes the refresh-token jti from Redis so /auth/refresh rejects it. Access tokens remain valid until their natural expiry — clients should discard them on logout.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      requests.UserRefreshRequest  true  "Refresh token to revoke"
// @Success      200      {object}  v1.BaseResponse  "Logged out"
// @Failure      401      {object}  v1.BaseResponse  "Refresh token invalid"
// @Failure      422      {object}  v1.BaseResponse  "Validation error"
// @Router       /auth/logout [post]
func (userH UserHandler) Logout(ctx *gin.Context) {
	var req requests.UserRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewErrorResponse(ctx, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validators.ValidatePayloads(req); err != nil {
		RespondWithError(ctx, err)
		return
	}

	if err := userH.usecase.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		RespondWithError(ctx, err)
		return
	}

	ev := auditFromGin(ctx)
	ev.Type = audit.EventLogout
	ev.Success = true
	audit.Record(ev)

	NewSuccessResponse(ctx, http.StatusOK, "logout success", nil)
}

// GetUserData godoc
// @Summary      Return the current user's profile
// @Description  Reads the authenticated user from the JWT in the Authorization header and returns the matching record (in-memory cache first, Postgres on miss).
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  v1.BaseResponse{data=responses.UserResponse}  "User profile"
// @Failure      401  {object}  v1.BaseResponse  "Missing or invalid token"
// @Failure      404  {object}  v1.BaseResponse  "User no longer exists"
// @Router       /users/me [get]
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
