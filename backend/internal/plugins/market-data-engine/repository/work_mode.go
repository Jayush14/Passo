package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// WorkModeRepository handles storage and retrieval of work modes
type WorkModeRepository struct {
	db *pgxpool.Pool
}

// NewWorkModeRepository creates a new WorkModeRepository
func NewWorkModeRepository(db *pgxpool.Pool) *WorkModeRepository {
	return &WorkModeRepository{db: db}
}

// Insert adds a new work mode and returns its ID
func (r *WorkModeRepository) Insert(ctx context.Context, workMode *models.WorkMode) (uuid.UUID, error) {
	query := `
		INSERT INTO work_modes
			(id, mode, normalized_mode, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		workMode.ID,
		workMode.Mode,
		workMode.NormalizedMode,
		workMode.TaxonomyVersion,
		workMode.CreatedAt,
		workMode.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a work mode by its ID
func (r *WorkModeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkMode, error) {
	query := `
		SELECT id, mode, normalized_mode, taxonomy_version, created_at, updated_at
		FROM work_modes
		WHERE id = $1
	`

	var workMode models.WorkMode
	err := r.db.QueryRow(ctx, query, id).Scan(
		&workMode.ID,
		&workMode.Mode,
		&workMode.NormalizedMode,
		&workMode.TaxonomyVersion,
		&workMode.CreatedAt,
		&workMode.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &workMode, nil
}

// GetByCanonicalName retrieves a work mode by its canonical name.
func (r *WorkModeRepository) GetByCanonicalName(ctx context.Context, canonicalName string) (*models.WorkMode, error) {
	query := `
		SELECT id, mode, normalized_mode, taxonomy_version, created_at, updated_at
		FROM work_modes
		WHERE mode = $1
	`

	var workMode models.WorkMode
	err := r.db.QueryRow(ctx, query, canonicalName).Scan(
		&workMode.ID,
		&workMode.Mode,
		&workMode.NormalizedMode,
		&workMode.TaxonomyVersion,
		&workMode.CreatedAt,
		&workMode.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &workMode, nil
}

// List returns all work modes
func (r *WorkModeRepository) List(ctx context.Context) ([]*models.WorkMode, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, mode, normalized_mode, taxonomy_version, created_at, updated_at
		FROM work_modes
		ORDER BY mode
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workModes []*models.WorkMode
	for rows.Next() {
		var workMode models.WorkMode
		if err := rows.Scan(
			&workMode.ID,
			&workMode.Mode,
			&workMode.NormalizedMode,
			&workMode.TaxonomyVersion,
			&workMode.CreatedAt,
			&workMode.UpdatedAt,
		); err != nil {
			return nil, err
		}
		workModes = append(workModes, &workMode)
	}

	return workModes, rows.Err()
}

// Update updates an existing work mode
func (r *WorkModeRepository) Update(ctx context.Context, workMode *models.WorkMode) error {
	query := `
		UPDATE work_modes
		SET mode = $2,
		    normalized_mode = $3,
		    taxonomy_version = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		workMode.ID,
		workMode.Mode,
		workMode.NormalizedMode,
		workMode.TaxonomyVersion,
		workMode.UpdatedAt,
	)

	return err
}

// Delete removes a work mode by its ID
func (r *WorkModeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM work_modes
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}