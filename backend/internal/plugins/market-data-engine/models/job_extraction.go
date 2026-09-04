package models

import (
	"time"
	"github.com/google/uuid"
)

// JobExtraction represents a raw job extraction from an external source
type JobExtraction struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	RawContent  string    `json:"raw_content"`
	Language    string    `json:"language"`
	CreatedAt   time.Time `json:"created_at"`
}