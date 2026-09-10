package models

import (
	"time"
	"github.com/google/uuid"
)

// JobSkill represents a skill associated with a job
type JobSkill struct {
	ID        uuid.UUID `json:"id"`
	JobID     uuid.UUID `json:"job_id"`
	SkillID   uuid.UUID `json:"skill_id"`
	RequirementType string `json:"requirement_type"` // required, preferred, nice_to_have
	RequirementWeight float64 `json:"requirement_weight"`
	Confidence    float64   `json:"confidence"`
	Evidence      string    `json:"evidence,omitempty"`
	RawSkillName  string    `json:"raw_skill_name"`
	NormalizationVersion uuid.UUID `json:"normalization_version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}