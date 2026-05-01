package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AdminConfig handles GET /admin/config — admin only
func AdminConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"config": map[string]interface{}{
			"max_incidents_per_page": 100,
			"default_severity":       3,
		},
	})
}
