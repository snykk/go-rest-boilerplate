package jwt

import (
	"errors"
	"time"

	golangJWT "github.com/golang-jwt/jwt/v5"
)

type JWTService interface {
	GenerateToken(userId string, isAdmin bool, email string) (t string, err error)
	ParseToken(tokenString string) (claims JwtCustomClaim, err error)
}

type JwtCustomClaim struct {
	UserID  string
	IsAdmin bool
	Email   string
	golangJWT.RegisteredClaims
}

type jwtService struct {
	secretKey string
	issuer    string
	expired   int
}

func NewJWTService(secretKey, issuer string, expired int) JWTService {
	return &jwtService{
		issuer:    issuer,
		secretKey: secretKey,
		expired:   expired,
	}
}

func (j *jwtService) GenerateToken(userID string, isAdmin bool, email string) (t string, err error) {
	claims := &JwtCustomClaim{
		userID,
		isAdmin,
		email,
		golangJWT.RegisteredClaims{
			ExpiresAt: golangJWT.NewNumericDate(time.Now().Add(time.Hour * time.Duration(j.expired))),
			Issuer:    j.issuer,
			IssuedAt:  golangJWT.NewNumericDate(time.Now()),
		},
	}
	token := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, claims)
	t, err = token.SignedString([]byte(j.secretKey))
	return
}

func (j *jwtService) ParseToken(tokenString string) (claims JwtCustomClaim, err error) {
	if token, err := golangJWT.ParseWithClaims(tokenString, &claims, func(token *golangJWT.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	}); err != nil || !token.Valid {
		return JwtCustomClaim{}, errors.New("token is not valid")
	}

	return
}
