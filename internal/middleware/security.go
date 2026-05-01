package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeaders sets hardened HTTP security headers on every response.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("Content-Security-Policy", "default-src 'self'")
			return next(c)
		}
	}
}

// CacheNoStore sets cache-control headers to prevent caching of sensitive responses.
func CacheNoStore() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			return next(c)
		}
	}
}
