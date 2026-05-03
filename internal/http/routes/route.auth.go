package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	authhandler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1/auth"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"golang.org/x/time/rate"
)

// authRoute wires the /auth/* group — register / login / OTP /
// refresh / logout. All endpoints are anonymous (no JWT required) so
// the group gets the rate limiter and the tight body cap.
type authRoute struct {
	handler        authhandler.Handler
	router         *gin.RouterGroup
	rateLimiter    gin.HandlerFunc
	authMiddleware gin.HandlerFunc
}

// NewAuthRoute builds the route module. The rate limiter is built per
// route module (rather than passed in) because the limit is a
// property of the auth surface — different groups would carry
// different limits. The auth middleware is shared with the users
// route so ChangePassword can reuse the same JWT validation.
func NewAuthRoute(router *gin.RouterGroup, authUC auth.Usecase, authMiddleware gin.HandlerFunc) *authRoute {
	// 5 requests per minute per IP for auth endpoints.
	authRateLimiter := middlewares.NewRateLimiter(rate.Limit(5.0/60.0), 5)
	return &authRoute{
		handler:        authhandler.NewHandler(authUC),
		router:         router,
		rateLimiter:    authRateLimiter.Middleware(),
		authMiddleware: authMiddleware,
	}
}

// Routes mounts the /auth group and its endpoints.
func (r *authRoute) Routes() {
	v1 := r.router.Group("/v1")
	authGrp := v1.Group("/auth")
	// Auth payloads are small JSON blobs — capping at 4 KiB blocks
	// slow-body / oversized-payload attacks against the only routes
	// that accept anonymous traffic, without affecting any
	// legitimate request.
	authGrp.Use(r.rateLimiter)
	authGrp.Use(middlewares.BodySizeLimitMiddleware(middlewares.AuthBodyMaxBytes))
	authGrp.POST("/register", r.handler.Register)
	authGrp.POST("/login", r.handler.Login)
	authGrp.POST("/send-otp", r.handler.SendOTP)
	authGrp.POST("/verify-otp", r.handler.VerifyOTP)
	authGrp.POST("/refresh", r.handler.Refresh)
	authGrp.POST("/logout", r.handler.Logout)
	authGrp.POST("/password/forgot", r.handler.ForgotPassword)
	authGrp.POST("/password/reset", r.handler.ResetPassword)
	// /password/change requires the JWT — gets the auth middleware
	// on top of the same rate limiter / body cap as the rest.
	authGrp.PUT("/password/change", r.authMiddleware, r.handler.ChangePassword)
}
