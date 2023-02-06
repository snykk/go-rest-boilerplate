package usecases

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
)

type userUsecase struct {
	repo domains.UserRepository
}

func NewUserUsecase(repo domains.UserRepository) domains.UserUsecase {
	return &userUsecase{
		repo: repo,
	}
}

func (userUC userUsecase) Store(ctx context.Context, inDom *domains.UserDomain) (outDom domains.UserDomain, statusCode int, err error) {
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

func (userUC userUsecase) Login(ctx context.Context, inDom *domains.UserDomain) (outDom domains.UserDomain, statusCode int, err error) {
	return
}

func (userUC userUsecase) SendOTP(ctx context.Context, email string) (otpCode string, statusCode int, err error) {
	return
}

func (userUC userUsecase) VerifOTP(ctx context.Context, email string, userOTP string, otpRedis string) (statusCode int, err error) {
	return
}
