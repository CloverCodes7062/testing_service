package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError returns a 422 response with a list of field errors.
func ValidationError(c echo.Context, errs []FieldError) error {
	return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
		"errors": errs,
	})
}

// ContainsHTML returns true if the string contains HTML-like characters.
func ContainsHTML(s string) bool {
	return strings.Contains(s, "<")
}
