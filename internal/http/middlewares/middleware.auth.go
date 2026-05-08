package middlewares

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	V1Handler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

type AuthMiddleware struct {
	jwtService jwt.JWTService
	redisCache caches.RedisCache
	isAdmin    bool
}

func NewAuthMiddleware(jwtService jwt.JWTService, redisCache caches.RedisCache, isAdmin bool) gin.HandlerFunc {
	return (&AuthMiddleware{
		jwtService: jwtService,
		redisCache: redisCache,
		isAdmin:    isAdmin,
	}).Handle
}

func (m *AuthMiddleware) Handle(ctx *gin.Context) {
	const (
		middlewareName = "AuthMiddleware"
		fileName       = "middleware.auth.go"
	)
	logCtx := ctx.Request.Context()

	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		logger.WarnWithContext(logCtx, "Auth: missing Authorization header", logger.Fields{
			"middleware": middlewareName,
			"file":       fileName,
			"step":       "read_header",
			"path":       ctx.Request.URL.Path,
		})
		V1Handler.NewAbortResponse(ctx, "missing authorization header")
		return
	}

	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		logger.WarnWithContext(logCtx, "Auth: invalid Authorization header format", logger.Fields{
			"middleware": middlewareName,
			"file":       fileName,
			"step":       "parse_header",
			"path":       ctx.Request.URL.Path,
		})
		V1Handler.NewAbortResponse(ctx, "invalid header format")
		return
	}

	if headerParts[0] != "Bearer" {
		logger.WarnWithContext(logCtx, "Auth: non-Bearer scheme", logger.Fields{
			"middleware": middlewareName,
			"file":       fileName,
			"step":       "scheme_check",
			"path":       ctx.Request.URL.Path,
			"scheme":     headerParts[0],
		})
		V1Handler.NewAbortResponse(ctx, "token must content bearer")
		return
	}

	user, err := m.jwtService.ParseToken(headerParts[1])
	if err != nil {
		logger.WarnWithContext(logCtx, "Auth: token parse failed", logger.Fields{
			"middleware": middlewareName,
			"file":       fileName,
			"step":       "parse_token",
			"path":       ctx.Request.URL.Path,
			"error":      err.Error(),
		})
		V1Handler.NewAbortResponse(ctx, "invalid token")
		return
	}

	// Reject access tokens issued before the user's most recent password
	// rotation. The cutoff is published to Redis by ChangePassword /
	// ResetPassword; absence (Redis miss) means no recent rotation, so
	// the token is allowed through. Redis errors fail open — the token
	// already passed signature + expiry, and we don't want a Redis blip
	// to lock everyone out.
	if m.redisCache != nil && user.IssuedAt != nil {
		if cutoffStr, getErr := m.redisCache.Get(logCtx, fmt.Sprintf("pwd_cutoff:%s", user.UserID)); getErr == nil && cutoffStr != "" {
			if cutoff, parseErr := strconv.ParseInt(cutoffStr, 10, 64); parseErr == nil && user.IssuedAt.Unix() < cutoff {
				logger.WarnWithContext(logCtx, "Auth: token revoked by password rotation", logger.Fields{
					"middleware": middlewareName,
					"file":       fileName,
					"step":       "check_pwd_cutoff",
					"path":       ctx.Request.URL.Path,
					"user_id":    user.UserID,
					"issued_at":  user.IssuedAt.Unix(),
					"cutoff":     cutoff,
				})
				V1Handler.NewAbortResponse(ctx, "token has been revoked")
				return
			}
		}
	}

	if user.IsAdmin != m.isAdmin && !user.IsAdmin {
		logger.WarnWithContext(logCtx, "Auth: insufficient privilege", logger.Fields{
			"middleware":     middlewareName,
			"file":           fileName,
			"step":           "privilege_check",
			"path":           ctx.Request.URL.Path,
			"user_id":        user.UserID,
			"required_admin": m.isAdmin,
			"user_is_admin":  user.IsAdmin,
		})
		V1Handler.NewAbortResponse(ctx, "you don't have access for this action")
		return
	}

	ctx.Set(constants.CtxAuthenticatedUserKey, user)
	ctx.Next()
}
