package guardrail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"passo/backend/internal/config"
)

// AuthMiddleware validates JWT from Authorization header and adds userID to context.
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "invalid Authorization format", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]

		// Parse token
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(config.LoadConfig().JWTSecret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}
		userIDFloat, ok := claims["user_id"]
		if !ok {
			http.Error(w, "missing user_id claim", http.StatusUnauthorized)
			return
		}
		var userID int64
		switch v := userIDFloat.(type) {
		case float64:
			userID = int64(v)
		case json.Number:
			val, _ := v.Int64()
			userID = val
		default:
			http.Error(w, "invalid user_id type", http.StatusUnauthorized)
			return
		}
		// Add userID to context
		const userIDKey = "userID"
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}