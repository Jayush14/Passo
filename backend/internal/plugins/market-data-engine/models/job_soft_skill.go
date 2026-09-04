package models

import (
	"time"
	"github.com/google/uuid"
)

// JobSoftSkill represents a soft skill associated with a job posting
type JobSoftSkill struct {
	ID           uuid.UUID `json:"id"`
	JobPostingID uuid.UUID `json:"job_posting_id"`
	SoftSkillID  uuid.UUID `json:"soft_skill_id"`
	Proficiency  string    `json:"proficiency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}