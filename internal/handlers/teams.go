package handlers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"testing_service/internal/db"
)

// CreateTeam handles POST /teams — manager+
func CreateTeam(c echo.Context) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	var errs []FieldError
	if body.Name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "name is required"})
	}
	if ContainsHTML(body.Name) {
		errs = append(errs, FieldError{Field: "name", Message: "HTML is not allowed"})
	}
	if len(errs) > 0 {
		return ValidationError(c, errs)
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	var team db.Team
	err := db.Pool.QueryRow(context.Background(),
		`INSERT INTO teams (name) VALUES ($1)
		 RETURNING id, name, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		body.Name,
	).Scan(&team.ID, &team.Name, &team.CreatedAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create team"})
	}

	return c.JSON(http.StatusCreated, team)
}
