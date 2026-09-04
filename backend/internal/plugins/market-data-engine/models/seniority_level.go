package models

import (
	"time"
	"github.com/google/uuid"
)

// SeniorityLevel represents a seniority level
type SeniorityLevel struct {
	ID   uuid.UUID `json:"id"`
	Level string    `json:"level"`
	NormalizedLevel string `json:"normalized_level"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}