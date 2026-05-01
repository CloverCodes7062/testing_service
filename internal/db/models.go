package db

type User struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	CreatedAt    string `json:"created_at"`
	PasswordHash string `json:"-"`
}

type Team struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type Incident struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Severity    int       `json:"severity"`
	TeamID      string    `json:"team_id"`
	Team        *Team     `json:"team,omitempty"`
	Comments    []Comment `json:"comments,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

type Comment struct {
	ID         string `json:"id"`
	IncidentID string `json:"incident_id"`
	UserID     string `json:"user_id"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuditLog struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	UserID     string `json:"user_id"`
	CreatedAt  string `json:"created_at"`
}
