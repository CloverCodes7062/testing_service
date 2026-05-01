package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"testing_service/internal/db"
)

// HealthHandler returns service status and DB pool statistics.
func HealthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "running",
		"pool":   db.PoolStats(),
	})
}
