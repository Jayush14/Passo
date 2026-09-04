package models

import (
	"time"
	"github.com/google/uuid"
)

// TaxonomyVersion represents a version of the taxonomy
type TaxonomyVersion struct {
	ID        uuid.UUID `json:"id"`
	Version   string    `json:"version"` // e.g., "1.0.0"
	Description string    `json:"description,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}