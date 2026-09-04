package models

import (
	"time"
	"github.com/google/uuid"
)

// Skill represents a skill in the taxonomy
type Skill struct {
	ID              uuid.UUID `json:"id"`
	CanonicalName   string    `json:"canonical_name"`
	NormalizedName  string    `json:"normalized_name"`
	SkillTypeID     uuid.UUID `json:"skill_type_id"`
	CategoryID      uuid.UUID `json:"category_id"`
	ParentSkillID   *uuid.UUID `json:"parent_skill_id"`
	Description     string    `json:"description,omitempty"`
	Status          string    `json:"status"`
	Version         int       `json:"version"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}