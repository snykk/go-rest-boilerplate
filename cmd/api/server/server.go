package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	V1Handler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/snykk/go-rest-boilerplate/internal/http/routes"
	"github.com/snykk/go-rest-boilerplate/internal/utils"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
)

type App struct {
	HttpServer *http.Server
	db         *sqlx.DB
	redisCache caches.RedisCache
}

func NewApp() (*App, error) {
	// setup databases
	conn, err := utils.SetupPostgresConnection()
	if err != nil {
		return nil, err
	}

	// setup router
	router := setupRouter()

	// jwt service
	jwtService := jwt.NewJWTService(config.AppConfig.JWTSecret, config.AppConfig.JWTIssuer, config.AppConfig.JWTExpired)

	// cache
	redisCache := caches.NewRedisCache(config.AppConfig.REDISHost, 0, config.AppConfig.REDISPassword, time.Duration(config.AppConfig.REDISExpired))
	ristrettoCache, err := caches.NewRistrettoCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	// mailer
	mailerService := mailer.NewOTPMailer(config.AppConfig.OTPEmail, config.AppConfig.OTPPassword)

	// auth middleware — user with valid token can access endpoint
	authMiddleware := middlewares.NewAuthMiddleware(jwtService, false)

	// Infrastructure endpoints (outside /api group)
	healthHandler := V1Handler.NewHealthHandler(conn, redisCache.Client())
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API Routes
	api := router.Group("api")
	api.GET("/", routes.RootHandler)
	routes.NewUsersRoute(api, conn, jwtService, redisCache, ristrettoCache, authMiddleware, mailerService).Routes()

	// setup http server
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.AppConfig.Port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return &App{
		HttpServer: server,
		db:         conn,
		redisCache: redisCache,
	}, nil
}

func (a *App) Run() (err error) {
	go func() {
		logger.InfoF("success to listen and serve on :%d", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryServer}, config.AppConfig.Port)
		if err := a.HttpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to listen and serve: %+v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("shutdown server ...", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryServer})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.HttpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("error when shutdown server: %v", err)
	}

	// close database connection
	if err := a.db.Close(); err != nil {
		logger.InfoF("error closing database: %v", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryServer}, err)
	}

	// close redis connection
	if err := a.redisCache.Close(); err != nil {
		logger.InfoF("error closing redis: %v", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryServer}, err)
	}

	logger.Info("server exiting", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryServer})
	return
}

func setupRouter() *gin.Engine {
	// set the runtime mode
	var mode = gin.ReleaseMode
	if config.AppConfig.Debug {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)

	// create a new router instance
	router := gin.New()

	// set up middlewares
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(middlewares.MetricsMiddleware())
	router.Use(middlewares.CORSMiddleware())
	router.Use(middlewares.BodySizeLimitMiddleware(1 << 20)) // 1MB max body size
	router.Use(gin.LoggerWithFormatter(logger.HTTPLogger))
	router.Use(gin.Recovery())

	return router
}
