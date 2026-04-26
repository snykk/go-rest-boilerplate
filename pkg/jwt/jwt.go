package jwt

import (
	"errors"
	"fmt"
	"time"

	golangJWT "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/snykk/go-rest-boilerplate/pkg/clock"
)

// ErrInvalidToken is returned when a token fails parsing or validation.
var ErrInvalidToken = errors.New("token is not valid")

// ErrWrongTokenKind is returned when a caller parses an access token
// as a refresh token, or vice versa.
var ErrWrongTokenKind = errors.New("token kind mismatch")

// Kinds distinguish access from refresh tokens. Signed into the claim
// so a compromised refresh token can't be reused as an access token.
const (
	KindAccess  = "access"
	KindRefresh = "refresh"
)

type JWTService interface {
	// GenerateToken mints a single access token. Kept for legacy call
	// sites; new code should prefer GenerateTokenPair.
	GenerateToken(userId string, isAdmin bool, email string) (t string, err error)
	// GenerateTokenPair mints an access+refresh pair, both signed with
	// the same secret but distinguished by the Kind claim.
	GenerateTokenPair(userID string, isAdmin bool, email string) (TokenPair, error)
	// ParseToken verifies the signature, expiry, and HMAC method of
	// an access token. Refresh tokens are rejected with ErrWrongTokenKind.
	ParseToken(tokenString string) (claims JwtCustomClaim, err error)
	// ParseRefreshToken is the refresh-token counterpart of ParseToken.
	// Access tokens are rejected with ErrWrongTokenKind.
	ParseRefreshToken(tokenString string) (claims JwtCustomClaim, err error)
}

// TokenPair bundles the short-lived access token and the long-lived
// refresh token issued together at login / refresh.
type TokenPair struct {
	AccessToken        string    `json:"access_token"`
	RefreshToken       string    `json:"refresh_token"`
	AccessExpiresAt    time.Time `json:"access_expires_at"`
	RefreshExpiresAt   time.Time `json:"refresh_expires_at"`
	AccessJTI          string    `json:"-"`
	RefreshJTI         string    `json:"-"`
}

type JwtCustomClaim struct {
	UserID  string
	IsAdmin bool
	Email   string
	Kind    string
	golangJWT.RegisteredClaims
}

type jwtService struct {
	secretKey      string
	issuer         string
	expired        int
	refreshExpired int // days
	// clock is the source of "now" used for IssuedAt and expiry
	// arithmetic. Default RealClock; tests inject clock.Frozen or
	// clock.Stub to assert exact timestamps without sleeping.
	clock clock.Clock
}

func NewJWTService(secretKey, issuer string, expired int) JWTService {
	return &jwtService{
		issuer:         issuer,
		secretKey:      secretKey,
		expired:        expired,
		refreshExpired: 7,
		clock:          clock.RealClock{},
	}
}

// NewJWTServiceWithRefresh is the constructor used by the server wiring
// once refresh-token support lands; the shorter NewJWTService stays so
// existing call sites and tests keep compiling.
func NewJWTServiceWithRefresh(secretKey, issuer string, expiredHours, refreshExpiredDays int) JWTService {
	return &jwtService{
		issuer:         issuer,
		secretKey:      secretKey,
		expired:        expiredHours,
		refreshExpired: refreshExpiredDays,
		clock:          clock.RealClock{},
	}
}

// WithClock returns a copy of the service with the given clock
// substituted. Tests use this to freeze time around token issuance
// so they can assert exact ExpiresAt values.
func WithClock(svc JWTService, c clock.Clock) JWTService {
	if s, ok := svc.(*jwtService); ok {
		copy := *s
		copy.clock = c
		return &copy
	}
	return svc
}

func (j *jwtService) GenerateToken(userID string, isAdmin bool, email string) (string, error) {
	tok, _, _, err := j.signAccess(userID, isAdmin, email)
	return tok, err
}

func (j *jwtService) GenerateTokenPair(userID string, isAdmin bool, email string) (TokenPair, error) {
	access, accessExp, accessJTI, err := j.signAccess(userID, isAdmin, email)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, refreshExp, refreshJTI, err := j.signRefresh(userID, email)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		AccessJTI:        accessJTI,
		RefreshJTI:       refreshJTI,
	}, nil
}

func (j *jwtService) signAccess(userID string, isAdmin bool, email string) (string, time.Time, string, error) {
	now := j.clock.Now()
	exp := now.Add(time.Hour * time.Duration(j.expired))
	jti := uuid.NewString()
	claims := &JwtCustomClaim{
		UserID:  userID,
		IsAdmin: isAdmin,
		Email:   email,
		Kind:    KindAccess,
		RegisteredClaims: golangJWT.RegisteredClaims{
			ID:        jti,
			ExpiresAt: golangJWT.NewNumericDate(exp),
			Issuer:    j.issuer,
			IssuedAt:  golangJWT.NewNumericDate(now),
		},
	}
	signed, err := j.sign(claims)
	if err != nil {
		return "", time.Time{}, "", err
	}
	return signed, exp, jti, nil
}

func (j *jwtService) signRefresh(userID, email string) (string, time.Time, string, error) {
	now := j.clock.Now()
	exp := now.Add(24 * time.Hour * time.Duration(j.refreshExpired))
	jti := uuid.NewString()
	claims := &JwtCustomClaim{
		UserID: userID,
		Email:  email,
		Kind:   KindRefresh,
		RegisteredClaims: golangJWT.RegisteredClaims{
			ID:        jti,
			ExpiresAt: golangJWT.NewNumericDate(exp),
			Issuer:    j.issuer,
			IssuedAt:  golangJWT.NewNumericDate(now),
		},
	}
	signed, err := j.sign(claims)
	if err != nil {
		return "", time.Time{}, "", err
	}
	return signed, exp, jti, nil
}

func (j *jwtService) sign(claims *JwtCustomClaim) (string, error) {
	token := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func (j *jwtService) ParseToken(tokenString string) (JwtCustomClaim, error) {
	claims, err := j.parse(tokenString)
	if err != nil {
		return JwtCustomClaim{}, err
	}
	// Tokens signed before the Kind field existed serialize with an
	// empty Kind — treat those as access to preserve backwards compat
	// during the rollout. New tokens carry KindAccess explicitly.
	if claims.Kind != "" && claims.Kind != KindAccess {
		return JwtCustomClaim{}, ErrWrongTokenKind
	}
	return claims, nil
}

func (j *jwtService) ParseRefreshToken(tokenString string) (JwtCustomClaim, error) {
	claims, err := j.parse(tokenString)
	if err != nil {
		return JwtCustomClaim{}, err
	}
	if claims.Kind != KindRefresh {
		return JwtCustomClaim{}, ErrWrongTokenKind
	}
	return claims, nil
}

func (j *jwtService) parse(tokenString string) (JwtCustomClaim, error) {
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
