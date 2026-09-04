package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// CapabilityRepository handles storage and retrieval of capabilities
type CapabilityRepository struct {
	db *pgxpool.Pool
}

// NewCapabilityRepository creates a new CapabilityRepository
func NewCapabilityRepository(db *pgxpool.Pool) *CapabilityRepository {
	return &CapabilityRepository{db: db}
}

// Insert adds a new capability and returns its ID
func (r *CapabilityRepository) Insert(ctx context.Context, capability *models.Capability) (uuid.UUID, error) {
	query := `
		INSERT INTO capabilities
			(id, name, category, description, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		capability.ID,
		capability.Name,
		capability.Category,
		capability.Description,
		capability.TaxonomyVersion,
		capability.CreatedAt,
		capability.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a capability by its ID
func (r *CapabilityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Capability, error) {
	query := `
		SELECT id, name, category, description, taxonomy_version, created_at, updated_at
		FROM capabilities
		WHERE id = $1
	`

	var capability models.Capability
	err := r.db.QueryRow(ctx, query, id).Scan(
		&capability.ID,
		&capability.Name,
		&capability.Category,
		&capability.Description,
		&capability.TaxonomyVersion,
		&capability.CreatedAt,
		&capability.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &capability, nil
}

// GetByName retrieves a capability by its name
func (r *CapabilityRepository) GetByName(ctx context.Context, name string) (*models.Capability, error) {
	query := `
		SELECT id, name, category, description, taxonomy_version, created_at, updated_at
		FROM capabilities
		WHERE name = $1
	`

	var capability models.Capability
	err := r.db.QueryRow(ctx, query, name).Scan(
		&capability.ID,
		&capability.Name,
		&capability.Category,
		&capability.Description,
		&capability.TaxonomyVersion,
		&capability.CreatedAt,
		&capability.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &capability, nil
}

// List returns all capabilities
func (r *CapabilityRepository) List(ctx context.Context) ([]*models.Capability, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, category, description, taxonomy_version, created_at, updated_at
		FROM capabilities
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var capabilities []*models.Capability
	for rows.Next() {
		var capability models.Capability
		if err := rows.Scan(
			&capability.ID,
			&capability.Name,
			&capability.Category,
			&capability.Description,
			&capability.TaxonomyVersion,
			&capability.CreatedAt,
			&capability.UpdatedAt,
		); err != nil {
			return nil, err
		}
		capabilities = append(capabilities, &capability)
	}

	return capabilities, rows.Err()
}

// Update updates an existing capability
func (r *CapabilityRepository) Update(ctx context.Context, capability *models.Capability) error {
	query := `
		UPDATE capabilities
		SET name = $2,
		    category = $3,
		    description = $4,
		    taxonomy_version = $5,
		    updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		capability.ID,
		capability.Name,
		capability.Category,
		capability.Description,
		capability.TaxonomyVersion,
		capability.UpdatedAt,
	)

	return err
}

// Delete removes a capability by its ID
func (r *CapabilityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM capabilities
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}