package models

import (
	"time"
	"github.com/google/uuid"
)

// SkillRelationship represents a relationship between two skills
type SkillRelationship struct {
	ID              uuid.UUID  `json:"id"`
	SourceSkillID   uuid.UUID  `json:"source_skill_id"`
	TargetSkillID   uuid.UUID  `json:"target_skill_id"`
	RelationshipType string     `json:"relationship_type"` // e.g., "required_for", "enhances"
	Strength        float64    `json:"strength"`
	TaxonomyVersion uuid.UUID  `json:"taxonomy_version"`
	CreatedAt       time.Time  `json:"created_at"`
}