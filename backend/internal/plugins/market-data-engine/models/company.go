package models

import (
	"time"
	"github.com/google/uuid"
)

// Company represents a company in the marketplace
type Company struct {
	ID              uuid.UUID `json:"id"`
	CanonicalName   string    `json:"canonical_name"`
	NormalizedName  string    `json:"normalized_name"`
	Website         string    `json:"website,omitempty"`
	Industry        string    `json:"industry,omitempty"`
	Size            string    `json:"size,omitempty"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}