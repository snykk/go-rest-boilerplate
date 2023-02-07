package usecases

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
)

type userUsecase struct {
	jwtService jwt.JWTService
	repo       domains.UserRepository
	mailer     mailer.OTPMailer
}

func NewUserUsecase(repo domains.UserRepository, jwtService jwt.JWTService, mailer mailer.OTPMailer) domains.UserUsecase {
	return &userUsecase{
		repo:       repo,
		jwtService: jwtService,
		mailer:     mailer,
	}
}

func (userUC *userUsecase) Store(ctx context.Context, inDom *domains.UserDomain) (outDom domains.UserDomain, statusCode int, err error) {
	inDom.Password, err = helpers.GenerateHash(inDom.Password)
	if err != nil {
		return domains.UserDomain{}, http.StatusInternalServerError, err
	}

	inDom.CreatedAt = time.Now().In(constants.GMT7)
	fmt.Println(time.Now().In(constants.GMT7))
	err = userUC.repo.Store(ctx, inDom)
	if err != nil {
		return domains.UserDomain{}, http.StatusInternalServerError, err
	}

	outDom, err = userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return domains.UserDomain{}, http.StatusInternalServerError, err
	}

	return outDom, http.StatusCreated, nil
}

func (userUC *userUsecase) Login(ctx context.Context, inDom *domains.UserDomain) (outDom domains.UserDomain, statusCode int, err error) {
	userDomain, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return domains.UserDomain{}, http.StatusUnauthorized, errors.New("invalid email or password") // for security purpose better use generic error message
	}

	if !userDomain.Active {
		return domains.UserDomain{}, http.StatusForbidden, errors.New("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return domains.UserDomain{}, http.StatusUnauthorized, errors.New("invalid email or password")
	}

	if userDomain.RoleID == constants.AdminID {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, true, userDomain.Email, userDomain.Password)
	} else {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, false, userDomain.Email, userDomain.Password)
	}

	if err != nil {
		return domains.UserDomain{}, http.StatusInternalServerError, err
	}

	return userDomain, http.StatusOK, nil
}

func (userUC *userUsecase) SendOTP(ctx context.Context, email string) (otpCode string, statusCode int, err error) {
	domain, err := userUC.repo.GetByEmail(ctx, &domains.UserDomain{Email: email})
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
	domain, err := userUC.repo.GetByEmail(ctx, &domains.UserDomain{Email: email})
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
	user, err := userUC.repo.GetByEmail(ctx, &domains.UserDomain{Email: email})
	if err != nil {
		return http.StatusNotFound, errors.New("email not found")
	}

	if err = userUC.repo.ChangeActiveUser(ctx, &domains.UserDomain{ID: user.ID, Active: true}); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
