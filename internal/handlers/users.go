package handlers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"testing_service/internal/db"
)

// ListUsers handles GET /users — admin only
func ListUsers(c echo.Context) error {
	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	rows, err := db.Pool.Query(context.Background(),
		`SELECT id, name, email, role, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM users ORDER BY created_at`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to query users"})
	}
	defer rows.Close()

	users := []db.User{}
	for rows.Next() {
		var u db.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan user"})
		}
		users = append(users, u)
	}

	return c.JSON(http.StatusOK, users)
}

// CreateUser handles POST /users — admin only
func CreateUser(c echo.Context) error {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	var errs []FieldError
	if body.Name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "name is required"})
	}
	if body.Email == "" {
		errs = append(errs, FieldError{Field: "email", Message: "email is required"})
	}
	if ContainsHTML(body.Name) || ContainsHTML(body.Email) || ContainsHTML(body.Role) {
		errs = append(errs, FieldError{Field: "input", Message: "HTML is not allowed in input fields"})
	}
	if len(errs) > 0 {
		return ValidationError(c, errs)
	}

	role := body.Role
	if role == "" {
		role = "member"
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	// Use a placeholder hash since no password is provided
	placeholder := "$2a$12$placeholder-hash-for-admin-created-user"

	var user db.User
	err := db.Pool.QueryRow(context.Background(),
		`INSERT INTO users (name, email, role, password_hash)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, email, role, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		body.Name, body.Email, role, placeholder,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		errStr := err.Error()
		if ContainsHTML(errStr) || strContains(errStr, "unique") || strContains(errStr, "duplicate") {
			return ValidationError(c, []FieldError{{Field: "email", Message: "email already in use"}})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	return c.JSON(http.StatusCreated, user)
}
