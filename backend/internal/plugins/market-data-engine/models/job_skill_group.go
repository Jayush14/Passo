package models

import (
	"time"
	"github.com/google/uuid"
)

// JobSkillGroup represents a group of skills for a job posting
type JobSkillGroup struct {
	ID          uuid.UUID `json:"id"`
	JobPostingID uuid.UUID `json:"job_posting_id"`
	GroupName   string    `json:"group_name"`
	SkillCount  int       `json:"skill_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}