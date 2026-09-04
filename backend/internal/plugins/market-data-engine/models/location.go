package models

import (
	"time"
	"github.com/google/uuid"
)

// Location represents a geographic location
type Location struct {
	ID          uuid.UUID `json:"id"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Country     string    `json:"country"`
	CountryCode string    `json:"country_code"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	TaxonomyVersion uuid.UUID `json:"taxonomy_version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}