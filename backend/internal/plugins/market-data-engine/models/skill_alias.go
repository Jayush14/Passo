package models

import (
	"time"
	"github.com/google/uuid"
)

// SkillAlias represents an alias for a skill
type SkillAlias struct {
	ID          uuid.UUID `json:"id"`
	SkillID     uuid.UUID `json:"skill_id"`
	Alias       string    `json:"alias"`
	NormalizedAlias string    `json:"normalized_alias"`
	Source      string    `json:"source"`
	Confidence  float64   `json:"confidence"`
	Status      string    `json:"status"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}