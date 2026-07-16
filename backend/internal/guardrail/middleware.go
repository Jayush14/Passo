package guardrail

import (
	"net/http"
	"time"
)

// Middleware is a simple wrapper that adds logging and can later include rate limiting, CORS, etc.
func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Call next handler
		next(w, r)
		// Log request (placeholder)
		_ = start
	}
}