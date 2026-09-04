package models

import (
	"github.com/google/uuid"
)

// JobDescription represents the description of a job posting
type JobDescription struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}
