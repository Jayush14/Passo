package subscription

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/config"
)

var cfg = config.LoadConfig()

// StartRequest represents a request to start a subscription (house‑key tier).
type StartRequest struct {
	// In future could include payment token, plan ID, etc.
	Plan string `json:"plan"` // e.g., "pro"
}

// StartResponse indicates success.
type StartResponse struct {
	Message string `json:"message"`
	Plan    string `json:"plan"`
}

// HandleStart activates the house‑key subscription for the authenticated user.
func HandleStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Plan == "" {
		http.Error(w, "plan required", http.StatusBadRequest)
		return
	}
	// TODO: Verify payment, set user.plan_tier = subscription, store house key reference.
	// For now, just respond OK.
	resp := StartResponse{
		Message: "subscription started",
		Plan:    req.Plan,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}