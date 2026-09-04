package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/lib/pq"
)

// DB wraps sql.DB.
type DB struct {
	*sql.DB
	Pool *pgxpool.Pool
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

	pool, err := pgxpool.New(context.Background(), dataSourceName)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = pool.Ping(context.Background()); err != nil {
		pool.Close()
		db.Close()
		return nil, err
	}

	return &DB{DB: db, Pool: pool}, nil
}

// Close releases both database connections.
func (db *DB) Close() error {
	if db.Pool != nil {
		db.Pool.Close()
	}
	return db.DB.Close()
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
		id           int64
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

// SaveAPIKey stores the encrypted API key for a user+provider (upsert).
func (db *DB) SaveAPIKey(ctx context.Context, userID int64, provider string, encryptedKey string) error {
	// Normalise provider (lowercase, trim) to avoid duplicates like "Gemini" vs "gemini"
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "gemini" // keep existing behaviour for old calls
	}
	_, err := db.ExecContext(
		ctx,
		`
		INSERT INTO api_keys (user_id, provider, encrypted_key, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, provider) DO UPDATE
			SET encrypted_key = EXCLUDED.encrypted_key,
				created_at    = NOW()
		`,
		userID, provider, encryptedKey,
	)
	return err
}

// GetAPIKey retrieves the encrypted API key for a user+provider.
// Returns sql.ErrNoRows if no row exists.
func (db *DB) GetAPIKey(ctx context.Context, userID int64, provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "gemini"
	}
	var encryptedKey string
	err := db.QueryRowContext(
		ctx,
		`SELECT encrypted_key FROM api_keys WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	).Scan(&encryptedKey)
	return encryptedKey, err // sql.ErrNoRows if not found
}

/* -------------------------------------------------------------------------
   Legacy wrappers – keep existing callers that do not pass a provider
   (they will keep using the Gemini key).  They can be removed later once
   all callers are updated.
   --------------------------------------------------------------------- */

// SaveAPIKeyLegacy preserves the old API (provider = "gemini").
func (db *DB) SaveAPIKeyLegacy(ctx context.Context, userID int64, encryptedKey string) error {
	return db.SaveAPIKey(ctx, userID, "gemini", encryptedKey)
}

// GetAPIKeyLegacy preserves the old API (provider = "gemini").
func (db *DB) GetAPIKeyLegacy(ctx context.Context, userID int64) (string, error) {
	return db.GetAPIKey(ctx, userID, "gemini")
}

// SaveRoadmap stores the generated roadmap JSON for a user.
func (db *DB) SaveRoadmap(ctx context.Context, userID int64, payload []byte) error {
	log.Printf("[db] SaveRoadmap called for user %d, payload size %d", userID, len(payload))
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO roadmaps (user_id, payload, created_at) VALUES ($1, $2, NOW())`,
		userID, payload,
	)
	if err != nil {
		log.Printf("[db] SaveRoadmap error: %v", err)
	}
	return err
}
