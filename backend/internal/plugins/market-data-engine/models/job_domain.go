package models

import (
	"time"
	"github.com/google/uuid"
)

// JobDomain represents a domain associated with a job posting
type JobDomain struct {
	ID           uuid.UUID `json:"id"`
	JobPostingID uuid.UUID `json:"job_posting_id"`
	DomainID     uuid.UUID `json:"domain_id"`
	Relevance    float64   `json:"relevance"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}