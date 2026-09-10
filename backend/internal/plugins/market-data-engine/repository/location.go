package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// LocationRepository handles storage and retrieval of locations
type LocationRepository struct {
	db *pgxpool.Pool
}

// NewLocationRepository creates a new LocationRepository
func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{db: db}
}

// Insert adds a new location and returns its ID
func (r *LocationRepository) Insert(ctx context.Context, location *models.Location) (uuid.UUID, error) {
	query := `
		INSERT INTO locations
			(id, city, state, country, country_code, lat, lon,
			 taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		location.ID,
		location.City,
		location.State,
		location.Country,
		location.CountryCode,
		location.Lat,
		location.Lon,
		location.TaxonomyVersion,
		location.CreatedAt,
		location.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a location by its ID
func (r *LocationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Location, error) {
	query := `
		SELECT id, city, state, country, country_code, lat, lon,
		       taxonomy_version, created_at, updated_at
		FROM locations
		WHERE id = $1
	`

	var location models.Location
	err := r.db.QueryRow(ctx, query, id).Scan(
		&location.ID,
		&location.City,
		&location.State,
		&location.Country,
		&location.CountryCode,
		&location.Lat,
		&location.Lon,
		&location.TaxonomyVersion,
		&location.CreatedAt,
		&location.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &location, nil
}

// GetByCityStateCountry retrieves a location by its city, state, and country.
func (r *LocationRepository) GetByCityStateCountry(ctx context.Context, city, state, country string) (*models.Location, error) {
	query := `
		SELECT id, city, state, country, country_code, lat, lon,
		       taxonomy_version, created_at, updated_at
		FROM locations
		WHERE city = $1 AND state = $2 AND country = $3
	`

	var location models.Location
	err := r.db.QueryRow(ctx, query, city, state, country).Scan(
		&location.ID,
		&location.City,
		&location.State,
		&location.Country,
		&location.CountryCode,
		&location.Lat,
		&location.Lon,
		&location.TaxonomyVersion,
		&location.CreatedAt,
		&location.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &location, nil
}

// List returns all locations
func (r *LocationRepository) List(ctx context.Context) ([]*models.Location, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, city, state, country, country_code, lat, lon,
		       taxonomy_version, created_at, updated_at
		FROM locations
		ORDER BY city, state, country
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []*models.Location
	for rows.Next() {
		var location models.Location
		if err := rows.Scan(
			&location.ID,
			&location.City,
			&location.State,
			&location.Country,
			&location.CountryCode,
			&location.Lat,
			&location.Lon,
			&location.TaxonomyVersion,
			&location.CreatedAt,
			&location.UpdatedAt,
		); err != nil {
			return nil, err
		}
		locations = append(locations, &location)
	}

	return locations, rows.Err()
}

// Update updates an existing location
func (r *LocationRepository) Update(ctx context.Context, location *models.Location) error {
	query := `
		UPDATE locations
		SET city = $2,
		    state = $3,
		    country = $4,
		    country_code = $5,
		    lat = $6,
		    lon = $7,
		    taxonomy_version = $8,
		    updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		location.ID,
		location.City,
		location.State,
		location.Country,
		location.CountryCode,
		location.Lat,
		location.Lon,
		location.TaxonomyVersion,
		location.UpdatedAt,
	)

	return err
}

// Delete removes a location by its ID
func (r *LocationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM locations
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}