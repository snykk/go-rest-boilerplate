package jwt

import (
	"errors"
	"fmt"
	"time"

	golangJWT "github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token fails parsing or validation.
var ErrInvalidToken = errors.New("token is not valid")

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

func (j *jwtService) GenerateToken(userID string, isAdmin bool, email string) (string, error) {
	claims := &JwtCustomClaim{
		UserID:  userID,
		IsAdmin: isAdmin,
		Email:   email,
		RegisteredClaims: golangJWT.RegisteredClaims{
			ExpiresAt: golangJWT.NewNumericDate(time.Now().Add(time.Hour * time.Duration(j.expired))),
			Issuer:    j.issuer,
			IssuedAt:  golangJWT.NewNumericDate(time.Now()),
		},
	}
	token := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func (j *jwtService) ParseToken(tokenString string) (JwtCustomClaim, error) {
	var claims JwtCustomClaim
	token, err := golangJWT.ParseWithClaims(tokenString, &claims, func(token *golangJWT.Token) (interface{}, error) {
		if _, ok := token.Method.(*golangJWT.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return JwtCustomClaim{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return JwtCustomClaim{}, ErrInvalidToken
	}
	return claims, nil
}
