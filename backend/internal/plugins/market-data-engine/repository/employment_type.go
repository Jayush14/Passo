package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// EmploymentTypeRepository handles storage and retrieval of employment types
type EmploymentTypeRepository struct {
	db *pgxpool.Pool
}

// NewEmploymentTypeRepository creates a new EmploymentTypeRepository
func NewEmploymentTypeRepository(db *pgxpool.Pool) *EmploymentTypeRepository {
	return &EmploymentTypeRepository{db: db}
}

// Insert adds a new employment type and returns its ID
func (r *EmploymentTypeRepository) Insert(ctx context.Context, employmentType *models.EmploymentType) (uuid.UUID, error) {
	query := `
		INSERT INTO employment_types
			(id, type, normalized_type, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		employmentType.ID,
		employmentType.Type,
		employmentType.NormalizedType,
		employmentType.TaxonomyVersion,
		employmentType.CreatedAt,
		employmentType.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves an employment type by its ID
func (r *EmploymentTypeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EmploymentType, error) {
	query := `
		SELECT id, type, normalized_type, taxonomy_version, created_at, updated_at
		FROM employment_types
		WHERE id = $1
	`

	var employmentType models.EmploymentType
	err := r.db.QueryRow(ctx, query, id).Scan(
		&employmentType.ID,
		&employmentType.Type,
		&employmentType.NormalizedType,
		&employmentType.TaxonomyVersion,
		&employmentType.CreatedAt,
		&employmentType.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &employmentType, nil
}

// GetByCanonicalName retrieves an employment type by its canonical name.
func (r *EmploymentTypeRepository) GetByCanonicalName(ctx context.Context, canonicalName string) (*models.EmploymentType, error) {
	query := `
		SELECT id, type, normalized_type, taxonomy_version, created_at, updated_at
		FROM employment_types
		WHERE type = $1
	`

	var employmentType models.EmploymentType
	err := r.db.QueryRow(ctx, query, canonicalName).Scan(
		&employmentType.ID,
		&employmentType.Type,
		&employmentType.NormalizedType,
		&employmentType.TaxonomyVersion,
		&employmentType.CreatedAt,
		&employmentType.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &employmentType, nil
}

// List returns all employment types
func (r *EmploymentTypeRepository) List(ctx context.Context) ([]*models.EmploymentType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, normalized_type, taxonomy_version, created_at, updated_at
		FROM employment_types
		ORDER BY type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employmentTypes []*models.EmploymentType
	for rows.Next() {
		var employmentType models.EmploymentType
		if err := rows.Scan(
			&employmentType.ID,
			&employmentType.Type,
			&employmentType.NormalizedType,
			&employmentType.TaxonomyVersion,
			&employmentType.CreatedAt,
			&employmentType.UpdatedAt,
		); err != nil {
			return nil, err
		}
		employmentTypes = append(employmentTypes, &employmentType)
	}

	return employmentTypes, rows.Err()
}

// Update updates an existing employment type
func (r *EmploymentTypeRepository) Update(ctx context.Context, employmentType *models.EmploymentType) error {
	query := `
		UPDATE employment_types
		SET type = $2,
		    normalized_type = $3,
		    taxonomy_version = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		employmentType.ID,
		employmentType.Type,
		employmentType.NormalizedType,
		employmentType.TaxonomyVersion,
		employmentType.UpdatedAt,
	)

	return err
}

// Delete removes an employment type by its ID
func (r *EmploymentTypeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM employment_types
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}