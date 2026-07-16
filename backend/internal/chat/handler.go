package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"sync"
	"time"

	"passo/backend/internal/apikey"
	"passo/backend/internal/db"
	"passo/backend/internal/gemini"
)

// ChatRequest represents the incoming chat message.
type ChatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model,omitempty"` // optional, defaults to gemini-2.0-flash
}

// ChatResponse represents the AI mentor's reply.
type ChatResponse struct {
	Reply string `json:"reply"`
}

// database is the shared DB instance for this package.
var database *db.DB

// SetDB sets the database instance for the chat handlers.
func SetDB(db *db.DB) {
	database = db
}

// idempotencyCache stores responses for duplicate request detection.
type idempotencyCache struct {
	mu      sync.Mutex
	entries map[string]entry
}

// entry holds a cached response and its expiration time.
type entry struct {
	response []byte
	expires  time.Time
}

var idempotencyCacheInstance = idempotencyCache{
	entries: make(map[string]entry),
}

// cleanup runs periodically to remove expired entries.
func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			idempotencyCacheInstance.mu.Lock()
			for k, v := range idempotencyCacheInstance.entries {
				if time.Now().After(v.expires) {
					delete(idempotencyCacheInstance.entries, k)
				}
			}
			idempotencyCacheInstance.mu.Unlock()
		}
	}()
}

// getOrCreate returns cached response if exists and not expired, otherwise returns nil, false.
func (c *idempotencyCache) getOrCreate(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent, ok := c.entries[key]; ok && time.Now().Before(ent.expires) {
		return ent.response, true
	}
	return nil, false
}

// store stores the response for the given key with a TTL of 10 seconds.
func (c *idempotencyCache) store(key string, response []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry{
		response: response,
		expires:  time.Now().Add(10 * time.Second),
	}
}

// HashRequest generates a hash for idempotency based on request attributes.
func hashRequest(r *http.Request, req ChatRequest) string {
	h := fnv.New32a()
	h.Write([]byte(r.RemoteAddr))
	h.Write([]byte(r.UserAgent()))
	h.Write([]byte(req.Message))
	h.Write([]byte(req.Model)) // include model in hash
	return fmt.Sprintf("%x", h.Sum32())
}

// HandleChat processes a chat message and returns a response from the Gemini API.
// It expects a JSON body with a "message" field and optional "model".
// It retrieves the user's encrypted API key from the database, decrypts it,
// calls the Gemini function, and returns the AI-generated reply.
func HandleChat(w http.ResponseWriter, r *http.Request) {
	// Decode request body
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		req.Model = "gemini-2.0-flash" // default model
	}

	// Idempotency-Key header: if present, use it for deduplication.
	// If not present, generate a key from request attributes.
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = hashRequest(r, req)
	}

	// Check if we have a recent response for this key.
	if cachedResp, found := idempotencyCacheInstance.getOrCreate(idempotencyKey); found {
		// Return cached response.
		w.Header().Set("Content-Type", "application/json")
		w.Write(cachedResp)
		return
	}

	// Retrieve user ID from context (set by auth middleware).
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get encrypted API key from database.
	if database == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}
	encryptedKey, err := database.GetAPIKey(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "API key not found for user", http.StatusUnauthorized)
			return
		}
		http.Error(w, "could not retrieve API key", http.StatusInternalServerError)
		return
	}

	// Decrypt the API key.
	apiKey, err := apikey.Decrypt(encryptedKey)
	if err != nil {
		http.Error(w, "decryption failed", http.StatusInternalServerError)
		return
	}

	fmt.Printf("User %d requested chat with model %s: %s\n", userID, req.Model, req.Message, apiKey) // Log the request for debugging purposes.
	// Call Gemini API.
	generatedText, err := gemini.GenerateText(apiKey, req.Message, req.Model)
	if err != nil {
		// Log the error for debugging.
		fmt.Printf("Error generating text: %v\n", err)
		http.Error(w, "failed to generate response", http.StatusInternalServerError)
		return
	}

	// Prepare response.
	response := ChatResponse{
		Reply: generatedText,
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	// Store response for idempotency.
	idempotencyCacheInstance.store(idempotencyKey, responseBytes)

	// Return response.
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}
