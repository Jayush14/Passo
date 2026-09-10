package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// SeniorityLevelRepository handles storage and retrieval of seniority levels
type SeniorityLevelRepository struct {
	db *pgxpool.Pool
}

// NewSeniorityLevelRepository creates a new SeniorityLevelRepository
func NewSeniorityLevelRepository(db *pgxpool.Pool) *SeniorityLevelRepository {
	return &SeniorityLevelRepository{db: db}
}

// Insert adds a new seniority level and returns its ID
func (r *SeniorityLevelRepository) Insert(ctx context.Context, seniorityLevel *models.SeniorityLevel) (uuid.UUID, error) {
	query := `
		INSERT INTO seniority_levels
			(id, level, normalized_level, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		seniorityLevel.ID,
		seniorityLevel.Level,
		seniorityLevel.NormalizedLevel,
		seniorityLevel.TaxonomyVersion,
		seniorityLevel.CreatedAt,
		seniorityLevel.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a seniority level by its ID
func (r *SeniorityLevelRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SeniorityLevel, error) {
	query := `
		SELECT id, level, normalized_level, taxonomy_version, created_at, updated_at
		FROM seniority_levels
		WHERE id = $1
	`

	var seniorityLevel models.SeniorityLevel
	err := r.db.QueryRow(ctx, query, id).Scan(
		&seniorityLevel.ID,
		&seniorityLevel.Level,
		&seniorityLevel.NormalizedLevel,
		&seniorityLevel.TaxonomyVersion,
		&seniorityLevel.CreatedAt,
		&seniorityLevel.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &seniorityLevel, nil
}

// GetByCanonicalName retrieves a seniority level by its canonical name.
func (r *SeniorityLevelRepository) GetByCanonicalName(ctx context.Context, canonicalName string) (*models.SeniorityLevel, error) {
	query := `
		SELECT id, level, normalized_level, taxonomy_version, created_at, updated_at
		FROM seniority_levels
		WHERE normalized_level = $1
	`

	var seniorityLevel models.SeniorityLevel
	err := r.db.QueryRow(ctx, query, canonicalName).Scan(
		&seniorityLevel.ID,
		&seniorityLevel.Level,
		&seniorityLevel.NormalizedLevel,
		&seniorityLevel.TaxonomyVersion,
		&seniorityLevel.CreatedAt,
		&seniorityLevel.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &seniorityLevel, nil
}

// List returns all seniority levels
func (r *SeniorityLevelRepository) List(ctx context.Context) ([]*models.SeniorityLevel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, level, normalized_level, taxonomy_version, created_at, updated_at
		FROM seniority_levels
		ORDER BY level
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seniorityLevels []*models.SeniorityLevel
	for rows.Next() {
		var seniorityLevel models.SeniorityLevel
		if err := rows.Scan(
			&seniorityLevel.ID,
			&seniorityLevel.Level,
			&seniorityLevel.NormalizedLevel,
			&seniorityLevel.TaxonomyVersion,
			&seniorityLevel.CreatedAt,
			&seniorityLevel.UpdatedAt,
		); err != nil {
			return nil, err
		}
		seniorityLevels = append(seniorityLevels, &seniorityLevel)
	}

	return seniorityLevels, rows.Err()
}

// Update updates an existing seniority level
func (r *SeniorityLevelRepository) Update(ctx context.Context, seniorityLevel *models.SeniorityLevel) error {
	query := `
		UPDATE seniority_levels
		SET level = $2,
		    normalized_level = $3,
		    taxonomy_version = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		seniorityLevel.ID,
		seniorityLevel.Level,
		seniorityLevel.NormalizedLevel,
		seniorityLevel.TaxonomyVersion,
		seniorityLevel.UpdatedAt,
	)

	return err
}

// Delete removes a seniority level by its ID
func (r *SeniorityLevelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM seniority_levels
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}