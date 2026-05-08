package middlewares

import (
	"math"
	"net/http"
	"strconv"
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

// rateLimitedResponse mirrors the v1.BaseResponse JSON shape. The
// middleware can't import the handlers package without a cycle, so the
// envelope is duplicated here — keep the field tags in sync.
type rateLimitedResponse struct {
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		writeRateLimitHeaders(c, rl.burst, limiter)

		if !limiter.Allow() {
			retryAfter := retryAfterSeconds(limiter)
			c.Writer.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, rateLimitedResponse{
				Status:    false,
				Message:   "too many requests, please try again later",
				RequestID: c.GetString(RequestIDHeader),
			})
			return
		}

		c.Next()
	}
}

// writeRateLimitHeaders advertises the limit, current remaining tokens,
// and the unix timestamp at which the bucket will be full again.
// Clients use these to back off without burning a 429 first.
func writeRateLimitHeaders(c *gin.Context, burst int, limiter *rate.Limiter) {
	tokens := limiter.Tokens()
	remaining := max(int(math.Floor(tokens)), 0)
	resetSeconds := 0
	if r := float64(limiter.Limit()); r > 0 {
		// Seconds until the bucket is full again from its current level.
		missing := float64(burst) - tokens
		if missing > 0 {
			resetSeconds = int(math.Ceil(missing / r))
		}
	}
	h := c.Writer.Header()
	h.Set("X-RateLimit-Limit", strconv.Itoa(burst))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Duration(resetSeconds)*time.Second).Unix(), 10))
}

// retryAfterSeconds computes the wait time before a single new token
// will be available. Rounded up to the next whole second since RFC 7231
// Retry-After only accepts integer seconds.
func retryAfterSeconds(limiter *rate.Limiter) int {
	r := float64(limiter.Limit())
	if r <= 0 {
		return 1
	}
	deficit := 1.0 - limiter.Tokens()
	if deficit <= 0 {
		return 0
	}
	return int(math.Ceil(deficit / r))
}
