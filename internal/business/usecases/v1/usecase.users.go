package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
)

type userUsecase struct {
	jwtService jwt.JWTService
	repo       V1Domains.UserRepository
	mailer     mailer.OTPMailer
}

func NewUserUsecase(repo V1Domains.UserRepository, jwtService jwt.JWTService, mailer mailer.OTPMailer) V1Domains.UserUsecase {
	return &userUsecase{
		repo:       repo,
		jwtService: jwtService,
		mailer:     mailer,
	}
}

func (userUC *userUsecase) Store(ctx context.Context, inDom *V1Domains.UserDomain) (outDom V1Domains.UserDomain, statusCode int, err error) {
	inDom.Password, err = helpers.GenerateHash(inDom.Password)
	if err != nil {
		return V1Domains.UserDomain{}, http.StatusInternalServerError, err
	}

	inDom.CreatedAt = time.Now().In(constants.GMT7)
	fmt.Println(time.Now().In(constants.GMT7))
	err = userUC.repo.Store(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, http.StatusInternalServerError, err
	}

	outDom, err = userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, http.StatusInternalServerError, err
	}

	return outDom, http.StatusCreated, nil
}

func (userUC *userUsecase) Login(ctx context.Context, inDom *V1Domains.UserDomain) (outDom V1Domains.UserDomain, statusCode int, err error) {
	userDomain, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, http.StatusUnauthorized, errors.New("invalid email or password") // for security purpose better use generic error message
	}

	if !userDomain.Active {
		return V1Domains.UserDomain{}, http.StatusForbidden, errors.New("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return V1Domains.UserDomain{}, http.StatusUnauthorized, errors.New("invalid email or password")
	}

	if userDomain.RoleID == constants.AdminID {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, true, userDomain.Email, userDomain.Password)
	} else {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, false, userDomain.Email, userDomain.Password)
	}

	if err != nil {
		return V1Domains.UserDomain{}, http.StatusInternalServerError, err
	}

	return userDomain, http.StatusOK, nil
}

func (userUC *userUsecase) SendOTP(ctx context.Context, email string) (otpCode string, statusCode int, err error) {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return "", http.StatusNotFound, errors.New("email not found")
	}

	if domain.Active {
		return "", http.StatusBadRequest, errors.New("account already activated")
	}

	code, err := helpers.GenerateOTPCode(6)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	if err = userUC.mailer.SendOTP(code, email); err != nil {
		return "", http.StatusInternalServerError, err
	}

	return code, http.StatusOK, nil
}

func (userUC *userUsecase) VerifOTP(ctx context.Context, email string, userOTP string, otpRedis string) (statusCode int, err error) {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return http.StatusNotFound, errors.New("email not found")
	}

	if domain.Active {
		return http.StatusBadRequest, errors.New("account already activated")
	}

	if otpRedis != userOTP {
		return http.StatusBadRequest, errors.New("invalid otp code")
	}

	return http.StatusOK, nil
}

func (userUC *userUsecase) ActivateUser(ctx context.Context, email string) (statusCode int, err error) {
	user, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return http.StatusNotFound, errors.New("email not found")
	}

	if err = userUC.repo.ChangeActiveUser(ctx, &V1Domains.UserDomain{ID: user.ID, Active: true}); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (uc *userUsecase) GetByEmail(ctx context.Context, email string) (outDom V1Domains.UserDomain, statusCode int, err error) {
	user, err := uc.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return V1Domains.UserDomain{}, http.StatusNotFound, errors.New("email not found")
	}

	return user, http.StatusOK, nil
}
