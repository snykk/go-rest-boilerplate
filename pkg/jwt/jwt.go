package jwt

import (
	"errors"
	"time"

	driJWT "github.com/dgrijalva/jwt-go"
)

type JWTService interface {
	GenerateToken(userId string, isAdmin bool, email string, password string) (t string, err error)
	ParseToken(tokenString string) (claims JwtCustomClaim, err error)
}

type JwtCustomClaim struct {
	UserID   string
	IsAdmin  bool
	Email    string
	Password string
	driJWT.StandardClaims
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

func (j *jwtService) GenerateToken(userID string, isAdmin bool, email string, password string) (t string, err error) {
	claims := &JwtCustomClaim{
		userID,
		isAdmin,
		email,
		password,
		driJWT.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(j.expired)).Unix(),
			Issuer:    j.issuer,
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := driJWT.NewWithClaims(driJWT.SigningMethodHS256, claims)
	t, err = token.SignedString([]byte(j.secretKey))
	return
}

func (j *jwtService) ParseToken(tokenString string) (claims JwtCustomClaim, err error) {
	if token, err := driJWT.ParseWithClaims(tokenString, &claims, func(token *driJWT.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	}); err != nil || !token.Valid {
		return JwtCustomClaim{}, errors.New("token is not valid")
	}

	return
}
