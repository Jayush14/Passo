package models

import (
	"time"
	"github.com/google/uuid"
)

// Domain represents a job domain/field
type Domain struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	ParentID        *uuid.UUID `json:"parent_id,omitempty"`
	Category        string     `json:"category"`
	TaxonomyVersion uuid.UUID  `json:"taxonomy_version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}