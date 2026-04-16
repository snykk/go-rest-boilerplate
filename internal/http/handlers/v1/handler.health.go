package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db          *sqlx.DB
	redisClient *redis.Client
}

func NewHealthHandler(db *sqlx.DB, redisClient *redis.Client) HealthHandler {
	return HealthHandler{db: db, redisClient: redisClient}
}

func (h HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "service is healthy",
	})
}

func (h HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	// check database
	if err := h.db.PingContext(ctx); err != nil {
		checks["database"] = "unreachable: " + err.Error()
		healthy = false
	} else {
		checks["database"] = "ok"
	}

	// check redis
	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unreachable: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": healthy,
		"checks": checks,
	})
}
