package models

import (
	"time"
	"github.com/google/uuid"
)

// JobTitle represents a job title
type JobTitle struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	NormalizedTitle string    `json:"normalized_title"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}