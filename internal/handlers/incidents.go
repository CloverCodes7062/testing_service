package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"testing_service/internal/auth"
	"testing_service/internal/db"
)

// ListIncidents handles GET /incidents — member+
func ListIncidents(c echo.Context) error {
	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	// Parse query params
	statusFilter := c.QueryParam("status")
	severityStr := c.QueryParam("severity")
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page := 1
	limit := 20
	if pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			page = v
		}
	}
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			if v > 100 {
				v = 100
			}
			limit = v
		}
	}
	offset := (page - 1) * limit

	query := `SELECT id, title, description, status, severity, COALESCE(team_id::text, ''), to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM incidents WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if statusFilter != "" {
		query += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, statusFilter)
		argIdx++
	}
	if severityStr != "" {
		if sev, err := strconv.Atoi(severityStr); err == nil {
			query += " AND severity = $" + strconv.Itoa(argIdx)
			args = append(args, sev)
			argIdx++
		}
	}
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to query incidents"})
	}
	defer rows.Close()

	incidents := []db.Incident{}
	for rows.Next() {
		var inc db.Incident
		if err := rows.Scan(&inc.ID, &inc.Title, &inc.Description, &inc.Status, &inc.Severity, &inc.TeamID, &inc.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan incident"})
		}
		incidents = append(incidents, inc)
	}

	return c.JSON(http.StatusOK, incidents)
}

// IncidentStats handles GET /incidents/stats — member+
func IncidentStats(c echo.Context) error {
	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	var total, open, investigating, resolved, closed int
	err := db.Pool.QueryRow(context.Background(),
		`SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'open'),
			COUNT(*) FILTER (WHERE status = 'investigating'),
			COUNT(*) FILTER (WHERE status = 'resolved'),
			COUNT(*) FILTER (WHERE status = 'closed')
		 FROM incidents`,
	).Scan(&total, &open, &investigating, &resolved, &closed)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to query stats"})
	}

	return c.JSON(http.StatusOK, map[string]int{
		"total":         total,
		"open":          open,
		"investigating": investigating,
		"resolved":      resolved,
		"closed":        closed,
	})
}

// GetIncident handles GET /incidents/:id — member+
func GetIncident(c echo.Context) error {
	id := c.Param("id")

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	var inc db.Incident
	var team db.Team
	var teamIDStr, teamName, teamCreatedAt *string

	err := db.Pool.QueryRow(context.Background(),
		`SELECT i.id, i.title, i.description, i.status, i.severity,
		        COALESCE(i.team_id::text, ''),
		        to_char(i.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        t.id::text, t.name, to_char(t.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM incidents i
		 LEFT JOIN teams t ON i.team_id = t.id
		 WHERE i.id = $1`, id,
	).Scan(&inc.ID, &inc.Title, &inc.Description, &inc.Status, &inc.Severity,
		&inc.TeamID, &inc.CreatedAt, &teamIDStr, &teamName, &teamCreatedAt)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	if teamIDStr != nil && *teamIDStr != "" {
		team.ID = *teamIDStr
		if teamName != nil {
			team.Name = *teamName
		}
		if teamCreatedAt != nil {
			team.CreatedAt = *teamCreatedAt
		}
		inc.Team = &team
	}

	// Load comments
	commentRows, err := db.Pool.Query(context.Background(),
		`SELECT id, incident_id, user_id, body, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM comments WHERE incident_id = $1 ORDER BY created_at`, id)
	if err == nil {
		defer commentRows.Close()
		for commentRows.Next() {
			var com db.Comment
			if err := commentRows.Scan(&com.ID, &com.IncidentID, &com.UserID, &com.Body, &com.CreatedAt); err == nil {
				inc.Comments = append(inc.Comments, com)
			}
		}
	}
	if inc.Comments == nil {
		inc.Comments = []db.Comment{}
	}

	return c.JSON(http.StatusOK, inc)
}

// CreateIncident handles POST /incidents — manager+
func CreateIncident(c echo.Context) error {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Severity    int    `json:"severity"`
		TeamID      string `json:"team_id"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	var errs []FieldError
	if body.Title == "" {
		errs = append(errs, FieldError{Field: "title", Message: "title is required"})
	}
	if len(errs) > 0 {
		return ValidationError(c, errs)
	}

	if body.Status == "" {
		body.Status = "open"
	}
	if body.Severity == 0 {
		body.Severity = 3
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	userID := auth.GetUserID(c)

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "transaction failed"})
	}
	defer tx.Rollback(context.Background())

	var inc db.Incident
	var teamIDPtr *string
	if body.TeamID != "" {
		teamIDPtr = &body.TeamID
	}

	err = tx.QueryRow(context.Background(),
		`INSERT INTO incidents (title, description, status, severity, team_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, title, description, status, severity, COALESCE(team_id::text, ''), to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		body.Title, body.Description, body.Status, body.Severity, teamIDPtr,
	).Scan(&inc.ID, &inc.Title, &inc.Description, &inc.Status, &inc.Severity, &inc.TeamID, &inc.CreatedAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create incident"})
	}

	// Insert audit log
	_, err = tx.Exec(context.Background(),
		`INSERT INTO audit_log (action, entity_type, entity_id, user_id) VALUES ($1, $2, $3, $4)`,
		"create", "incident", inc.ID, userID,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to write audit log"})
	}

	if err := tx.Commit(context.Background()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "transaction commit failed"})
	}

	return c.JSON(http.StatusCreated, inc)
}
