package models

import (
	"time"
	"github.com/google/uuid"
)

// WorkMode represents a work mode (remote, hybrid, on-site)
type WorkMode struct {
	ID   uuid.UUID `json:"id"`
	Mode string    `json:"mode"`
	NormalizedMode string `json:"normalized_mode"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}