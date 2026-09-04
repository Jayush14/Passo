package models

import (
	"time"
	"github.com/google/uuid"
)

// Job represents a job in the system
type Job struct {
	ID              uuid.UUID `json:"id"`
	ExternalID      string    `json:"external_id"`
	Source          string    `json:"source"`
	SourceURL       string    `json:"source_url"`
	RawContent      string    `json:"raw_content"`
	DedupKey        string    `json:"dedup_key"`
	Language        string    `json:"language"`
	RetrievedAt     time.Time `json:"retrieved_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	MDEProcessed    bool      `json:"mde_processed"`
	MDEProcessedAt  *string   `json:"mde_processed_at,omitempty"`
	MDEErrorMessage *string   `json:"mde_error_message,omitempty"`
}