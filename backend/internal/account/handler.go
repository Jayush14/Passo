package account

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/config"
)

var cfg = config.LoadConfig()

// HandleDelete deletes the user's account and all associated data.
func HandleDelete(w http.ResponseWriter, r *http.Request) {
	// TODO: authenticate user, delete rows from all tables.
	// For now, respond OK.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "account deletion not yet implemented"})
}

// HandleExport returns a JSON export of the user's data (GDPR).
func HandleExport(w http.ResponseWriter, r *http.Request) {
	// TODO: authenticate user, gather data.
	// For now, return empty structure.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":          nil,
		"roadmaps":      []interface{}{},
		"progress":      []interface{}{},
		"chat_messages": []interface{}{},
		"api_key":       nil,
	})
}