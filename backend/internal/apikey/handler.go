package apikey

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"crypto/rand"
	"net/http"
	"strings"
	"database/sql"

	"passo/backend/internal/config"
	"passo/backend/internal/db"
)

var (
	database *db.DB
	cfg      = config.LoadConfig()
)

// SetDB sets the database instance for the API key handlers.
func SetDB(db *db.DB) {
	database = db
}

// StoreKeyRequest represents the request to store an API key.
type StoreKeyRequest struct {
	Key           string `json:"key,omitempty"`           // plaintext API key
	EncryptedKey  string `json:"encrypted_key,omitempty"` // optional pre‑encrypted key (base64)
	Provider      string `json:"provider,omitempty"`      // e.g. "gemini", "huggingface", "groq", ...
}

// StoreKeyResponse is the response after storing a key.
type StoreKeyResponse struct {
	Message string `json:"message"`
}

// GetKeyRequest represents the request to retrieve an API key.
type GetKeyRequest struct {
	Provider string `json:"provider,omitempty"` // defaults to "gemini"
}

// GetKeyResponse is returned when retrieving the stored key.
type GetKeyResponse struct {
	Key string `json:"key"` // plaintext API key (returned for convenience)
}

// encrypt encrypts a plaintext string using AES‑256‑GCM with the configured key.
// The result is a base64‑encoded string containing: nonce || ciphertext.
func encrypt(plaintext string) (string, error) {
	// Decode the base64‑encoded encryption key from config.
	keyBytes, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode encryption key: %w", err)
	}
	if len(keyBytes) != 32 {
		return "", errors.New("encryption key must be 32 bytes after decoding")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64‑encoded ciphertext produced by encrypt.
func Decrypt(ciphertextB64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode encryption key: %w", err)
	}
	if len(keyBytes) != 32 {
		return "", errors.New("encryption key must be 32 bytes after decoding")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aesgcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := ciphertext[:aesgcm.NonceSize()]
	ciphertextOnly := ciphertext[aesgcm.NonceSize():]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertextOnly, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}
	return string(plaintext), nil
}

// normalizeProvider returns a canonical provider name (lowercase, trimmed),
// defaulting to "gemini" for empty input.
func normalizeProvider(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if p == "" {
		return "gemini"
	}
	return p
}

// HandleStoreKey stores the API key for the authenticated user.
// It accepts either a plaintext "key" (which it encrypts) or a pre‑encrypted
// "encrypted_key". The encrypted value is saved in the api_keys table.
func HandleStoreKey(w http.ResponseWriter, r *http.Request) {
	var req StoreKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var plaintext string
	if req.Key != "" {
		plaintext = req.Key
	} else if req.EncryptedKey != "" {
		// Assume the client already provided an encrypted key.
		plaintext = req.EncryptedKey
	} else {
		http.Error(w, "key or encrypted_key required", http.StatusBadRequest)
		return
	}

	var encrypted string
	var err error
	if req.Key != "" {
		// Encrypt the provided plaintext key.
		encrypted, err = encrypt(plaintext)
		if err != nil {
			http.Error(w, "encryption failed", http.StatusInternalServerError)
			return
		}
	} else {
		// Use the provided encrypted key as‑is (should be base64).
		encrypted = plaintext
		// Optional: verify it’s valid base64.
		if _, err := base64.StdEncoding.DecodeString(encrypted); err != nil {
			http.Error(w, "encrypted_key must be base64‑encoded", http.StatusBadRequest)
			return
		}
	}

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

	provider := normalizeProvider(req.Provider)

	if err := database.SaveAPIKey(r.Context(), userID, provider, encrypted); err != nil {
		http.Error(w, "could not store API key", http.StatusInternalServerError)
		return
	}

	resp := StoreKeyResponse{Message: "key stored"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetKey retrieves the stored API key for the authenticated user.
// It returns the plaintext key for convenience (encrypted in storage).
func HandleGetKey(w http.ResponseWriter, r *http.Request) {
	// Optional: allow provider via query param or JSON body.
	// We'll check query first, then fallback to body, then default.
	var req GetKeyRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // ignore error; empty body ok
	}
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

	provider := normalizeProvider(r.URL.Query().Get("provider"))
	if provider == "" {
		provider = normalizeProvider(req.Provider)
	}

	encrypted, err := database.GetAPIKey(r.Context(), userID, provider)
	if err != nil {
		if err == sql.ErrNoRows {
			// No key stored for this provider – return empty string as before
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(GetKeyResponse{Key: ""})
			return
		}
		http.Error(w, "could not retrieve API key", http.StatusInternalServerError)
		return
	}

	plaintext, err := Decrypt(encrypted)
	if err != nil {
		http.Error(w, "decryption failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetKeyResponse{Key: plaintext})
}