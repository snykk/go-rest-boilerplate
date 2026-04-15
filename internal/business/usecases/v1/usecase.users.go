package v1

import (
	"context"
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

func (userUC *userUsecase) Store(ctx context.Context, inDom *V1Domains.UserDomain) (outDom V1Domains.UserDomain, err error) {
	inDom.Password, err = helpers.GenerateHash(inDom.Password)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	inDom.CreatedAt = time.Now().In(constants.GMT7)
	err = userUC.repo.Store(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	outDom, err = userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	return outDom, nil
}

func (userUC *userUsecase) Login(ctx context.Context, inDom *V1Domains.UserDomain) (outDom V1Domains.UserDomain, err error) {
	userDomain, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrUnauthorized("invalid email or password")
	}

	if !userDomain.Active {
		return V1Domains.UserDomain{}, constants.ErrForbidden("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return V1Domains.UserDomain{}, constants.ErrUnauthorized("invalid email or password")
	}

	if userDomain.RoleID == constants.AdminID {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, true, userDomain.Email)
	} else {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, false, userDomain.Email)
	}

	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	return userDomain, nil
}

func (userUC *userUsecase) SendOTP(ctx context.Context, email string) (otpCode string, err error) {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return "", constants.ErrNotFound("email not found")
	}

	if domain.Active {
		return "", constants.ErrBadRequest("account already activated")
	}

	code, err := helpers.GenerateOTPCode(6)
	if err != nil {
		return "", constants.ErrInternal(err.Error())
	}

	if err = userUC.mailer.SendOTP(code, email); err != nil {
		return "", constants.ErrInternal(err.Error())
	}

	return code, nil
}

func (userUC *userUsecase) VerifOTP(ctx context.Context, email string, userOTP string, otpRedis string) error {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return constants.ErrNotFound("email not found")
	}

	if domain.Active {
		return constants.ErrBadRequest("account already activated")
	}

	if otpRedis != userOTP {
		return constants.ErrBadRequest("invalid otp code")
	}

	return nil
}

func (userUC *userUsecase) ActivateUser(ctx context.Context, email string) error {
	user, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return constants.ErrNotFound("email not found")
	}

	if err = userUC.repo.ChangeActiveUser(ctx, &V1Domains.UserDomain{ID: user.ID, Active: true}); err != nil {
		return constants.ErrInternal(err.Error())
	}

	return nil
}

func (userUC *userUsecase) GetByEmail(ctx context.Context, email string) (outDom V1Domains.UserDomain, err error) {
	user, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrNotFound("email not found")
	}

	return user, nil
}
