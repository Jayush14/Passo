package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps sql.DB.
type DB struct {
	*sql.DB
}

// New creates a new DB instance from a postgres connection string.
func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

// CreateUser inserts a new user and returns the user ID.
func (db *DB) CreateUser(ctx context.Context, email, passwordHash string) (int64, error) {
	var id int64
	err := db.QueryRowContext(
		ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		email, passwordHash,
	).Scan(&id)
	return id, err
}

// GetUserByEmail retrieves a user by email.
func (db *DB) GetUserByEmail(ctx context.Context, email string) (int64, string, error) {
	var (
		id          int64
		passwordHash string
	)
	err := db.QueryRowContext(
		ctx,
		"SELECT id, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&id, &passwordHash)
	return id, passwordHash, err // sql.ErrNoRows if not found
}

// SaveRefreshToken stores a SHA-256 hash of the refresh token for the user.
func (db *DB) SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	// Hash the token using SHA-256
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, hashStr, expiresAt,
	)
	return err
}

// GetUserIDFromRefreshToken returns the user ID associated with a valid refresh token.
func (db *DB) GetUserIDFromRefreshToken(ctx context.Context, token string) (int64, bool, error) {
	// Compute SHA-256 hash of the provided token
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])
	var userID int64
	var expiresAt time.Time
	err := db.QueryRowContext(
		ctx,
		`SELECT user_id, expires_at FROM refresh_tokens
		 WHERE token_hash = $1 AND expires_at > NOW()`,
		hashStr,
	).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, false, nil // token not found or expired
	}
	if err != nil {
		return 0, false, err
	}
	return userID, true, nil
}

// DeleteRefreshToken removes a specific refresh token (e.g., on logout).
func (db *DB) DeleteRefreshToken(ctx context.Context, userID int64, token string) error {
	// Compute SHA-256 hash of the token
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])
	_, err := db.ExecContext(
		ctx,
		"DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2",
		userID, hashStr,
	)
	return err
}

// SaveAPIKey stores the encrypted API key for the user (upsert).
func (db *DB) SaveAPIKey(ctx context.Context, userID int64, encryptedKey string) error {
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO api_keys (user_id, encrypted_key, created_at)
			 VALUES ($1, $2, NOW())
			 ON CONFLICT (user_id) DO UPDATE SET
			     encrypted_key = EXCLUDED.encrypted_key`,
		userID, encryptedKey,
	)
	return err
}

// GetAPIKey retrieves the encrypted API key for the user.
func (db *DB) GetAPIKey(ctx context.Context, userID int64) (string, error) {
	var encryptedKey string
	err := db.QueryRowContext(
		ctx,
		"SELECT encrypted_key FROM api_keys WHERE user_id = $1",
		userID,
	).Scan(&encryptedKey)
	return encryptedKey, err // sql.ErrNoRows if not found
}