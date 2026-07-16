package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"passo/backend/internal/config"
	"passo/backend/internal/db"
)

var (
	database *db.DB
	cfg      = config.LoadConfig()
)

// SetDB sets the database instance for auth handlers.
func SetDB(db *db.DB) {
	database = db
}

// SignUpRequest represents the request body for sign up.
type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignUpResponse represents the response.
type SignUpResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// HandleSignUp registers a new user.
func HandleSignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Insert user into DB
	userID, err := database.CreateUser(r.Context(), req.Email, string(hashed))
	if err != nil {
		// Handle duplicate email etc.
		http.Error(w, "could not create user", http.StatusInternalServerError)
		return
	}
	// Generate tokens
	accessToken, err := generateJWT(userID, cfg.JWTSecret, cfg.TokenExpiry)
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		http.Error(w, "could not generate refresh token", http.StatusInternalServerError)
		return
	}
	// Store refresh token hash with expiration (7 days)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := database.SaveRefreshToken(r.Context(), userID, refreshToken, expiresAt); err != nil {
		http.Error(w, "could not store refresh token", http.StatusInternalServerError)
		return
	}
	resp := SignUpResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// LoginRequest represents login credentials.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse returns tokens.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// HandleLogin authenticates a user and returns tokens.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	// Retrieve user by email
	userID, hashedPW, err := database.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		// If user not found or password mismatch, give generic error to avoid user enumeration.
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPW), []byte(req.Password))
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// Generate tokens
	accessToken, err := generateJWT(userID, cfg.JWTSecret, cfg.TokenExpiry)
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		http.Error(w, "could not generate refresh token", http.StatusInternalServerError)
		return
	}
	// Store refresh token hash with expiration (7 days)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := database.SaveRefreshToken(r.Context(), userID, refreshToken, expiresAt); err != nil {
		http.Error(w, "could not store refresh token", http.StatusInternalServerError)
		return
	}
	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RefreshRequest holds the refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse returns new access token and new refresh token (rotation).
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// HandleRefresh validates the refresh token and issues new access (+refresh) tokens.
func HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh token required", http.StatusBadRequest)
		return
	}
	// Validate refresh token and get user ID
	userID, valid, err := database.GetUserIDFromRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}
	// Optional: delete the used token to enforce one-time use (rotation)
	if err := database.DeleteRefreshToken(r.Context(), userID, req.RefreshToken); err != nil {
		// Not critical, but we can log
		_ = err
	}
	// Generate new tokens
	accessToken, err := generateJWT(userID, cfg.JWTSecret, cfg.TokenExpiry)
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		http.Error(w, "could not generate refresh token", http.StatusInternalServerError)
		return
	}
	// Store new refresh token
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := database.SaveRefreshToken(r.Context(), userID, newRefreshToken, expiresAt); err != nil {
		http.Error(w, "could not store refresh token", http.StatusInternalServerError)
		return
	}
	resp := RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// LogoutResponse is the JSON body we return.
type LogoutResponse struct {
	Message string `json:"message"`
}

// HandleLogout invalidates the refresh token for the authenticated user.
// Expects JSON body: { "refresh_token": "<token>" }
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token required", http.StatusBadRequest)
		return
	}
	// Retrieve user ID from context (set by AuthMiddleware)
	const userCtxKey = "userID"
	userIDI := r.Context().Value(userCtxKey)
	if userIDI == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDI.(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// Log for debugging
	log.Printf("Logout: userID=%v, refreshToken=%q", userID, req.RefreshToken)
	// Delete the refresh token row (hash computed inside the DB method)
	if err := database.DeleteRefreshToken(r.Context(), userID, req.RefreshToken); err != nil {
		// Not fatal from UX perspective; still consider logout successful.
		// Log the error for debugging if desired.
		_ = err
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LogoutResponse{Message: "logged out"})
}

// generateJWT creates a signed JWT token with user ID claim.
func generateJWT(userID int64, secret string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(expiry).Unix(),
		"iat":     time.Now().Unix(),
		"iss":     "passo",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// generateRefreshToken returns a cryptographically random URL‑safe token.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
