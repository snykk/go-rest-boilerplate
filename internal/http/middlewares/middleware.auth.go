package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	V1Handler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
)

type AuthMiddleware struct {
	jwtService jwt.JWTService
	isAdmin    bool
}

func NewAuthMiddleware(jwtService jwt.JWTService, isAdmin bool) gin.HandlerFunc {
	return (&AuthMiddleware{
		jwtService: jwtService,
		isAdmin:    isAdmin,
	}).Handle
}

func (m *AuthMiddleware) Handle(ctx *gin.Context) {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		V1Handler.NewAbortResponse(ctx, "missing authorization header")
		return
	}

	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		V1Handler.NewAbortResponse(ctx, "invalid header format")
		return
	}

	if headerParts[0] != "Bearer" {
		V1Handler.NewAbortResponse(ctx, "token must content bearer")
		return
	}

	user, err := m.jwtService.ParseToken(headerParts[1])
	if err != nil {
		V1Handler.NewAbortResponse(ctx, "invalid token")
		return
	}

	if user.IsAdmin != m.isAdmin && !user.IsAdmin {
		V1Handler.NewAbortResponse(ctx, "you don't have access for this action")
		return
	}

	ctx.Set(constants.CtxAuthenticatedUserKey, user)
	ctx.Next()
}
