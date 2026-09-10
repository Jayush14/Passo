package models

import (
	"time"
	"github.com/google/uuid"
)

// JobSoftSkill represents a soft skill associated with a job
type JobSoftSkill struct {
	ID         uuid.UUID `json:"id"`
	JobID      uuid.UUID `json:"job_id"`
	SoftSkillID uuid.UUID `json:"soft_skill_id"`
	Proficiency  string    `json:"proficiency,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}