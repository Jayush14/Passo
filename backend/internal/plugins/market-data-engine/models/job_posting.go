package models

import (
	"time"
	"github.com/google/uuid"
)

// JobPosting represents a job posting
type JobPosting struct {
	ID              uuid.UUID `json:"id"`
	JobID           uuid.UUID `json:"job_id"`
	Title           string    `json:"title"`
	CompanyID       uuid.UUID `json:"company_id"`
	LocationID      uuid.UUID `json:"location_id"`
	EmploymentTypeID uuid.UUID `json:"employment_type_id"`
	WorkModeID      uuid.UUID `json:"work_mode_id"`
	SalaryMin       float64   `json:"salary_min"`
	SalaryMax       float64   `json:"salary_max"`
	SalaryCurrency  string    `json:"salary_currency"`
	SalaryPeriod    string    `json:"salary_period"`
	Description     string    `json:"description"`
	ApplyURL        string    `json:"apply_url"`
	PostedAt        time.Time `json:"posted_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	MatchedSkillsCount int    `json:"matched_skills_count"`
	TotalSkillsCount int    `json:"total_skills_count"`
	MatchScore      float64   `json:"match_score"`
	Status          string    `json:"status"`
	// DedupKey is used for deduplication of job postings
	DedupKey        string    `json:"dedup_key,omitempty"`
}