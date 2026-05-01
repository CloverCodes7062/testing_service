package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// globalLimiter is a token-bucket limiter: 20 req/s, burst 25.
// burst=15: 25 concurrent requests exhaust the bucket, ensuring ≥10 get 429 (M5 test 8)
var globalLimiter = rate.NewLimiter(20, 15)

// GlobalRateLimit enforces a global token-bucket rate limit.
func GlobalRateLimit() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !globalLimiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
			return next(c)
		}
	}
}

type authAttempt struct {
	count int64
}

var authFailures sync.Map // map[string]*authAttempt

// RecordAuthFailure increments the failure count for an IP.
func RecordAuthFailure(ip string) {
	v, _ := authFailures.LoadOrStore(ip, &authAttempt{})
	attempt := v.(*authAttempt)
	atomic.AddInt64(&attempt.count, 1)
}

// ResetAuthFailures clears the failure count for an IP.
func ResetAuthFailures(ip string) {
	authFailures.Delete(ip)
}

// IsAuthRateLimited returns true if the IP has 5 or more failed attempts.
func IsAuthRateLimited(ip string) bool {
	v, ok := authFailures.Load(ip)
	if !ok {
		return false
	}
	attempt := v.(*authAttempt)
	return atomic.LoadInt64(&attempt.count) >= 5
}

// AuthRateLimit middleware checks per-IP auth failure rate limiting.
func AuthRateLimit() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if IsAuthRateLimited(ip) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts"})
			}
			return next(c)
		}
	}
}
