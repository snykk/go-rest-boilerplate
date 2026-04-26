package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	V1PostgresRepository "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/postgres/v1"
	V1Handler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"golang.org/x/time/rate"
)

type usersRoutes struct {
	V1Handler      V1Handler.UserHandler
	router         *gin.RouterGroup
	db             *sqlx.DB
	authMiddleware gin.HandlerFunc
	rateLimiter    gin.HandlerFunc
}

func NewUsersRoute(router *gin.RouterGroup, db *sqlx.DB, jwtService jwt.JWTService, redisCache caches.RedisCache, ristrettoCache caches.RistrettoCache, authMiddleware gin.HandlerFunc, mailer mailer.OTPMailer) *usersRoutes {
	V1UserRepository := V1PostgresRepository.NewUserRepository(db)
	V1UserUsecase := usecases.NewUserUsecase(V1UserRepository, jwtService, mailer, redisCache, ristrettoCache)
	V1UserHandler := V1Handler.NewUserHandler(V1UserUsecase)

	// 5 requests per minute per IP for auth endpoints
	authRateLimiter := middlewares.NewRateLimiter(rate.Limit(5.0/60.0), 5)

	return &usersRoutes{V1Handler: V1UserHandler, router: router, db: db, authMiddleware: authMiddleware, rateLimiter: authRateLimiter.Middleware()}
}

func (r *usersRoutes) Routes() {
	// Routes V1
	V1Route := r.router.Group("/v1")
	{
		// auth (rate limited + tight body cap)
		// Auth payloads are small JSON blobs — capping at 4 KiB blocks
		// slow-body / oversized-payload attacks against the only routes
		// that accept anonymous traffic, without affecting any
		// legitimate request.
		V1AuhtRoute := V1Route.Group("/auth")
		V1AuhtRoute.Use(r.rateLimiter)
		V1AuhtRoute.Use(middlewares.BodySizeLimitMiddleware(middlewares.AuthBodyMaxBytes))
		V1AuhtRoute.POST("/register", r.V1Handler.Register)
		V1AuhtRoute.POST("/login", r.V1Handler.Login)
		V1AuhtRoute.POST("/send-otp", r.V1Handler.SendOTP)
		V1AuhtRoute.POST("/verify-otp", r.V1Handler.VerifyOTP)
		V1AuhtRoute.POST("/refresh", r.V1Handler.Refresh)
		V1AuhtRoute.POST("/logout", r.V1Handler.Logout)

		// users
		userRoute := V1Route.Group("/users")
		userRoute.Use(r.authMiddleware)
		{
			userRoute.GET("/me", r.V1Handler.GetUserData)
			// ...
		}
	}
}
