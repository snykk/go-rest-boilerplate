package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*ipLimiter
	rate     rate.Limit
	burst    int
	stop     chan struct{}
}

func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*ipLimiter),
		rate:     r,
		burst:    burst,
		stop:     make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup removes stale entries every 3 minutes until Stop is called.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 5*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// Stop terminates the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"status":  false,
				"message": "too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}
