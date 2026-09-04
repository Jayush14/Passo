package models

import (
	"time"
	"github.com/google/uuid"
)

// JobSkill represents a skill associated with a job posting
type JobSkill struct {
	ID        uuid.UUID `json:"id"`
	JobPostingID uuid.UUID `json:"job_posting_id"`
	SkillID   uuid.UUID `json:"skill_id"`
	SkillLevel string    `json:"skill_level"`
	Importance float64   `json:"importance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}