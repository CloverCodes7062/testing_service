package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RoleLevel returns the numeric level for a role.
// admin=3, manager=2, member=1
func RoleLevel(role string) int {
	switch role {
	case "admin":
		return 3
	case "manager":
		return 2
	case "member":
		return 1
	default:
		return 0
	}
}

// SignToken creates a signed HS256 JWT with sub, role, and exp claims.
func SignToken(secret, userID, role string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates and parses a JWT, returning userID and role.
func ParseToken(secret, tokenStr string) (userID string, role string, err error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", "", errors.New("missing sub claim")
	}

	r, ok := claims["role"].(string)
	if !ok {
		return "", "", errors.New("missing role claim")
	}

	return sub, r, nil
}
