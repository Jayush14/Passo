package models

import (
	"time"
	"github.com/google/uuid"
)

// JobFamily represents a family of jobs
type JobFamily struct {
	ID   uuid.UUID `json:"id"`
	Family string    `json:"family"`
	NormalizedFamily string `json:"normalized_family"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}