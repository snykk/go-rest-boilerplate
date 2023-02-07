package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/http/handlers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
)

type usersRoutes struct {
	handlers       handlers.UserHandler
	router         *gin.Engine
	db             *sqlx.DB
	authMiddleware gin.HandlerFunc
}

func NewUsersRoute(router *gin.Engine, db *sqlx.DB, jwtService jwt.JWTService, redisCache caches.RedisCache, ristrettoCache caches.RistrettoCache, authMiddleware gin.HandlerFunc) *usersRoutes {
	userRepository := postgres.NewUserRepository(db)
	userUsecase := usecases.NewUserUsecase(userRepository, jwtService)
	UserHandler := handlers.NewUserHandler(userUsecase, redisCache, ristrettoCache)

	return &usersRoutes{handlers: UserHandler, router: router, db: db, authMiddleware: authMiddleware}
}

func (r *usersRoutes) Routes() {
	// auth
	auhtRoute := r.router.Group("auth")
	auhtRoute.POST("/regis", r.handlers.Regis)
	auhtRoute.POST("/login", r.handlers.Login)
	auhtRoute.POST("/send-otp", r.handlers.SendOTP)
	auhtRoute.POST("/verif-otp", r.handlers.VerifOTP)

}
