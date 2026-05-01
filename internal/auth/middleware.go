package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// RequireAuth returns middleware enforcing authentication and a minimum role.
//
// M3 mode: when MILESTONE_JWT_VALIDATE_URL is set, the bearer token is forwarded
// to the PWA validate endpoint and the returned role is used for RBAC checks.
// This is intentional — M3 tests middleware wiring, not JWT implementation.
//
// M4+ mode: when JWT_SECRET is set (and no validate URL), tokens are verified
// locally via HS256. The test runner mints tokens directly from JWT_SECRET.
func RequireAuth(minRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			var userID, role string

			if validateURL := os.Getenv("MILESTONE_JWT_VALIDATE_URL"); validateURL != "" {
				// M3: proxy validation to PWA — tokens are PWA-signed, student doesn't hold the secret
				uid, r, err := validateViaProxy(validateURL, tokenStr)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				}
				userID, role = uid, r
			} else {
				// M4+: local HS256 verification using the sandbox-injected JWT_SECRET
				secret := os.Getenv("JWT_SECRET")
				if secret == "" {
					secret = "default-secret"
				}
				uid, r, err := ParseToken(secret, tokenStr)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				}
				userID, role = uid, r
			}

			if RoleLevel(role) < RoleLevel(minRole) {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
			}

			c.Set("userID", userID)
			c.Set("userRole", role)
			return next(c)
		}
	}
}

// validateViaProxy forwards the bearer token to the PWA's validate-token endpoint
// and returns the subject (userID) and role from the response.
func validateViaProxy(validateURL, token string) (userID, role string, err error) {
	req, err := http.NewRequest(http.MethodGet, validateURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("validate proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Valid bool   `json:"valid"`
		Sub  string `json:"sub"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("invalid response from validate endpoint")
	}
	if !result.Valid {
		return "", "", fmt.Errorf("token is not valid")
	}
	return result.Sub, result.Role, nil
}

// GetUserID retrieves the authenticated user's ID from the echo context.
func GetUserID(c echo.Context) string {
	v, _ := c.Get("userID").(string)
	return v
}

// GetUserRole retrieves the authenticated user's role from the echo context.
func GetUserRole(c echo.Context) string {
	v, _ := c.Get("userRole").(string)
	return v
}
