package models

import (
	"time"
	"github.com/google/uuid"
)

// EmploymentType represents a type of employment
type EmploymentType struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
	NormalizedType string `json:"normalized_type"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}