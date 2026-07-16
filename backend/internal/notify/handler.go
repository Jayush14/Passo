package notify

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/config"
)

var cfg = config.LoadConfig()

// SubscribeRequest represents a push subscription from the browser.
type SubscribeRequest struct {
	Endpoint string          `json:"endpoint"`
	Keys     map[string]string `json:"keys"` // should contain p256dh and auth
}

// SubscribeResponse is the acknowledgment.
type SubscribeResponse struct {
	Message string `json:"message"`
}

// HandleSubscribe stores the push subscription for a user.
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Endpoint == "" || len(req.Keys) == 0 {
		http.Error(w, "endpoint and keys required", http.StatusBadRequest)
		return
	}
	// TODO: Save subscription associated with authenticated user.
	// For now, just acknowledge.
	resp := SubscribeResponse{Message: "subscription stored"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleVAPIDPublicKey returns the VAPID public key for the frontend.
func HandleVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	// In a real app, we would return the actual VAPID public key.
	// For dev, return empty string or a dummy.
	publicKey := cfg.VAPIDPublicKey
	if publicKey == "" {
		publicKey = "dummy-public-key"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"public_key": publicKey})
}