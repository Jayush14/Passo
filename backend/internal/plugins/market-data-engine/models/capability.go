package models

import (
	"time"
	"github.com/google/uuid"
)

// Capability represents a capability (certification, license, etc.)
type Capability struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Description     string    `json:"description"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}