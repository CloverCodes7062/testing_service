package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"testing_service/internal/auth"
	"testing_service/internal/db"
)

func jwtSecret() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "default-secret"
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func strContains(s, sub string) bool {
	return strings.Contains(s, sub)
}

// ---- Reset password rate limiting (per email) ----

type resetAttempt struct {
	count int64
}

var resetAttempts sync.Map // map[string]*resetAttempt

func recordResetAttempt(email string) {
	v, _ := resetAttempts.LoadOrStore(email, &resetAttempt{})
	attempt := v.(*resetAttempt)
	atomic.AddInt64(&attempt.count, 1)
}

func isResetRateLimited(email string) bool {
	v, ok := resetAttempts.Load(email)
	if !ok {
		return false
	}
	attempt := v.(*resetAttempt)
	return atomic.LoadInt64(&attempt.count) >= 3
}

// ---- Per-IP auth failure tracking (used by Login handler) ----

type authAttemptLocal struct {
	count int64
}

var (
	localAuthFailures sync.Map // map[string]*authAttemptLocal
)

func recordAuthFailureLocal(ip string) {
	v, _ := localAuthFailures.LoadOrStore(ip, &authAttemptLocal{})
	attempt := v.(*authAttemptLocal)
	atomic.AddInt64(&attempt.count, 1)
}

func resetAuthFailuresLocal(ip string) {
	localAuthFailures.Delete(ip)
}

func isAuthRateLimitedLocal(ip string) bool {
	v, ok := localAuthFailures.Load(ip)
	if !ok {
		return false
	}
	attempt := v.(*authAttemptLocal)
	return atomic.LoadInt64(&attempt.count) >= 5
}

// ---- Handlers ----

// Register handles POST /auth/register
func Register(c echo.Context) error {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
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
	if len(body.Password) < 8 {
		errs = append(errs, FieldError{Field: "password", Message: "password must be at least 8 characters"})
	}
	if ContainsHTML(body.Name) || ContainsHTML(body.Email) || ContainsHTML(body.Password) || ContainsHTML(body.Role) {
		errs = append(errs, FieldError{Field: "input", Message: "HTML is not allowed in input fields"})
	}
	if len(errs) > 0 {
		return ValidationError(c, errs)
	}

	role := body.Role
	if role == "" {
		role = "member"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	var user db.User
	err = db.Pool.QueryRow(context.Background(),
		`INSERT INTO users (name, email, role, password_hash)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, email, role, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		body.Name, body.Email, role, string(hash),
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		errStr := err.Error()
		if strContains(errStr, "unique") || strContains(errStr, "duplicate") {
			return ValidationError(c, []FieldError{{Field: "email", Message: "email already in use"}})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	return c.JSON(http.StatusCreated, user)
}

// Login handles POST /auth/login
func Login(c echo.Context) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	ip := c.RealIP()

	if isAuthRateLimitedLocal(ip) {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts"})
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	var user db.User
	err := db.Pool.QueryRow(context.Background(),
		`SELECT id, name, email, role, password_hash, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM users WHERE email = $1`,
		body.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		recordAuthFailureLocal(ip)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		recordAuthFailureLocal(ip)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	resetAuthFailuresLocal(ip)

	accessToken, err := auth.SignToken(jwtSecret(), user.ID, user.Role, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to sign access token"})
	}

	rawRefresh, err := generateToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate refresh token"})
	}

	refreshHash := hashToken(rawRefresh)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err = db.Pool.Exec(context.Background(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		user.ID, refreshHash, expiresAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store refresh token"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": rawRefresh,
	})
}

// Refresh handles POST /auth/refresh
func Refresh(c echo.Context) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	tokenHash := hashToken(body.RefreshToken)

	var userID string
	var revoked bool
	var expiresAt time.Time
	err := db.Pool.QueryRow(context.Background(),
		`SELECT user_id, revoked, expires_at FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&userID, &revoked, &expiresAt)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}

	if revoked || time.Now().After(expiresAt) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "refresh token expired or revoked"})
	}

	var role string
	err = db.Pool.QueryRow(context.Background(),
		`SELECT role FROM users WHERE id = $1`, userID,
	).Scan(&role)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
	}

	accessToken, err := auth.SignToken(jwtSecret(), userID, role, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
	}

	return c.JSON(http.StatusOK, map[string]string{"access_token": accessToken})
}

// ResetPasswordRequest handles POST /auth/reset-password/request
func ResetPasswordRequest(c echo.Context) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	successMsg := map[string]string{"message": "if that email exists, a reset link was sent"}

	if isResetRateLimited(body.Email) {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many reset requests"})
	}
	recordResetAttempt(body.Email)

	if db.Pool == nil {
		return c.JSON(http.StatusOK, successMsg)
	}

	var userID string
	err := db.Pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, body.Email,
	).Scan(&userID)
	if err != nil {
		return c.JSON(http.StatusOK, successMsg)
	}

	rawToken, err := generateToken()
	if err != nil {
		return c.JSON(http.StatusOK, successMsg)
	}

	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().Add(1 * time.Hour)

	_, _ = db.Pool.Exec(context.Background(),
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)

	return c.JSON(http.StatusOK, successMsg)
}

// ResetPasswordConfirm handles POST /auth/reset-password/confirm
func ResetPasswordConfirm(c echo.Context) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if len(body.NewPassword) < 8 {
		return ValidationError(c, []FieldError{{Field: "new_password", Message: "password must be at least 8 characters"}})
	}

	if db.Pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database not available"})
	}

	tokenHash := hashToken(body.Token)

	var tokenID string
	var userID string
	var used bool
	var expiresAt time.Time
	err := db.Pool.QueryRow(context.Background(),
		`SELECT id, user_id, used, expires_at FROM password_reset_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&tokenID, &userID, &used, &expiresAt)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
	}

	if used {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "token already used"})
	}
	if time.Now().After(expiresAt) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token expired"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "transaction failed"})
	}
	defer tx.Rollback(context.Background())

	if _, err = tx.Exec(context.Background(),
		`UPDATE password_reset_tokens SET used = TRUE WHERE id = $1`, tokenID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to mark token used"})
	}

	if _, err = tx.Exec(context.Background(),
		`UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
	}

	if _, err = tx.Exec(context.Background(),
		`UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1`, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to revoke tokens"})
	}

	if err := tx.Commit(context.Background()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "transaction commit failed"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "password updated"})
}
