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
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/drivers"
	userspostgres "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/postgres/users"
	V1Handler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/http/middlewares"
	"github.com/snykk/go-rest-boilerplate/internal/http/routes"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	HttpServer     *http.Server
	db             *sqlx.DB
	redisCache     caches.RedisCache
	asyncMailer    *mailer.AsyncOTPMailer
	tracerShutdown observability.Shutdown
}

func NewApp() (*App, error) {
	// Tracer first so any span emitted by later setup (DB connect,
	// migration check, etc.) lands in the right provider.
	shutdownTracer, err := observability.SetupTracing(context.Background(), observability.TracingConfig{
		ServiceName: "go-rest-boilerplate",
		Environment: config.AppConfig.Environment,
		Exporter:    config.AppConfig.OTelExporter,
		SampleRatio: config.AppConfig.OTelSampleRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("setup tracing: %w", err)
	}

	// setup databases
	conn, err := drivers.SetupSQLXPostgres()
	if err != nil {
		return nil, err
	}
	// Expose live DB pool stats via /metrics.
	observability.RegisterDBStatsProvider(func() *sqlx.DB { return conn })

	// setup router
	router := setupRouter()

	// jwt service
	jwtService := jwt.NewJWTServiceWithRefresh(
		config.AppConfig.JWTSecret,
		config.AppConfig.JWTIssuer,
		config.AppConfig.JWTExpired,
		config.AppConfig.JWTRefreshExpired,
	)

	// cache
	redisCache := caches.NewRedisCache(config.AppConfig.REDISHost, 0, config.AppConfig.REDISPassword, time.Duration(config.AppConfig.REDISExpired))
	ristrettoCache, err := caches.NewRistrettoCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	// mailer — wrap the synchronous SMTP sender in an async queue so
	// OTP send latency stays off the HTTP request path.
	syncMailer := mailer.NewOTPMailer(config.AppConfig.OTPEmail, config.AppConfig.OTPPassword)
	asyncMailer := mailer.NewAsyncOTPMailer(
		syncMailer,
		config.AppConfig.MailerWorkers,
		config.AppConfig.MailerQueueSize,
		config.AppConfig.MailerRetries,
		time.Second,
	)

	// auth middleware — user with valid token can access endpoint
	authMiddleware := middlewares.NewAuthMiddleware(jwtService, false)

	// Infrastructure endpoints (outside /api group)
	healthHandler := V1Handler.NewHealthHandler(conn, redisCache.Client())
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger UI — spec is generated from godoc annotations by
	// `make swag` (which runs `swag init -g cmd/api/main.go`). The
	// _ "<module>/docs" import in main.go registers the spec at init
	// time, so this route just needs to point at it.
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Compose the bounded contexts. Users owns identity CRUD, Auth
	// owns credential / session flows; Auth depends on Users for
	// reads/writes of user records.
	userRepo := userspostgres.NewUserRepository(conn)
	usersUC := users.NewUsecase(userRepo, ristrettoCache, users.Config{
		BcryptCost: config.AppConfig.BcryptCost,
	})
	authUC := auth.NewUsecase(usersUC, jwtService, asyncMailer, redisCache, auth.Config{
		OTPMaxAttempts:   config.AppConfig.OTPMaxAttempts,
		OTPTTL:           time.Duration(config.AppConfig.REDISExpired) * time.Minute,
		PasswordResetTTL: 30 * time.Minute,
		BcryptCost:       config.AppConfig.BcryptCost,
	})

	// API Routes
	api := router.Group("api")
	api.GET("/", routes.RootHandler)
	routes.NewAuthRoute(api, authUC, authMiddleware).Routes()
	routes.NewUsersRoute(api, usersUC, authMiddleware).Routes()

	// setup http server
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.AppConfig.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return &App{
		HttpServer:     server,
		db:             conn,
		redisCache:     redisCache,
		asyncMailer:    asyncMailer,
		tracerShutdown: shutdownTracer,
	}, nil
}

func (a *App) Run() (err error) {
	srvLog := logger.WithFields(logger.Fields{constants.LoggerCategory: constants.LoggerCategoryServer})

	go func() {
		srvLog.Infof("success to listen and serve on :%d", config.AppConfig.Port)
		if err := a.HttpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to listen and serve: %+v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	srvLog.Info("shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.HttpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("error when shutdown server: %v", err)
	}

	// drain the async mailer queue before tearing down dependencies so
	// in-flight OTP emails still get delivered.
	if a.asyncMailer != nil {
		if err := a.asyncMailer.Shutdown(ctx); err != nil {
			srvLog.Infof("mailer shutdown incomplete: %v", err)
		}
	}

	// close database connection
	if err := a.db.Close(); err != nil {
		srvLog.Infof("error closing database: %v", err)
	}

	// close redis connection
	if err := a.redisCache.Close(); err != nil {
		srvLog.Infof("error closing redis: %v", err)
	}

	// flush any spans the batch exporter is still buffering — must run
	// after the HTTP server stops accepting requests but before the
	// process exits, otherwise the tail end of in-flight traces is lost.
	if a.tracerShutdown != nil {
		if err := a.tracerShutdown(ctx); err != nil {
			srvLog.Infof("tracer shutdown incomplete: %v", err)
		}
	}

	srvLog.Info("server exiting")
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
	// otelgin first so the OTel span context (trace_id) is established
	// before RequestIDMiddleware reaches in to bridge it into the
	// logger context. Spans emitted further down the stack (DB, Redis,
	// mailer) become children of the server span automatically.
	router.Use(otelgin.Middleware("go-rest-boilerplate"))
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(middlewares.MetricsMiddleware())
	router.Use(middlewares.SecurityHeadersMiddleware())
	router.Use(middlewares.CORSMiddleware())
	router.Use(middlewares.BodySizeLimitMiddleware(middlewares.DefaultBodyMaxBytes))
	router.Use(gin.LoggerWithFormatter(middlewares.AccessLogFormatter))
	router.Use(gin.Recovery())

	return router
}
