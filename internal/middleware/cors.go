package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

var allowedOrigins = map[string]bool{
	"http://localhost:3001": true,
	"http://localhost:3000": true,
}

// CustomCORS handles CORS with a strict allowlist — never returns wildcard origin.
func CustomCORS() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")

			if allowedOrigins[origin] {
				c.Response().Header().Set("Access-Control-Allow-Origin", origin)
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Response().Header().Set("Vary", "Origin")
			}

			// Handle preflight
			if c.Request().Method == http.MethodOptions {
				if allowedOrigins[origin] {
					return c.NoContent(http.StatusNoContent)
				}
				return c.NoContent(http.StatusForbidden)
			}

			return next(c)
		}
	}
}
