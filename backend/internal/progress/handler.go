package progress

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/config"
)

var cfg = config.LoadConfig()

// UpdateTaskRequest represents the PATCH payload.
type UpdateTaskRequest struct {
	Checked bool   `json:"checked,omitempty"`
	Note    string `json:"note,omitempty"`
}

// HandleUpdateTask updates a task's checked status or note.
func HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	// Extract task_id from URL path
	path := r.URL.Path
	// In a real router we would extract the variable; for now ignore.
	_ = path

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	// TODO: persist to DB
	response := map[string]interface{}{
		"task_id":   "dummy",
		"checked":   req.Checked,
		"note":      req.Note,
		"updated_at": "2026-07-12T00:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}