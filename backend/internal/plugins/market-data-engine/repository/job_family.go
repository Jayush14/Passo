package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobFamilyRepository handles storage and retrieval of job families
type JobFamilyRepository struct {
	db *pgxpool.Pool
}

// NewJobFamilyRepository creates a new JobFamilyRepository
func NewJobFamilyRepository(db *pgxpool.Pool) *JobFamilyRepository {
	return &JobFamilyRepository{db: db}
}

// Insert adds a new job family and returns its ID
func (r *JobFamilyRepository) Insert(ctx context.Context, jobFamily *models.JobFamily) (uuid.UUID, error) {
	query := `
		INSERT INTO job_families
			(id, family, normalized_family, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobFamily.ID,
		jobFamily.Family,
		jobFamily.NormalizedFamily,
		jobFamily.TaxonomyVersion,
		jobFamily.CreatedAt,
		jobFamily.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job family by its ID
func (r *JobFamilyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobFamily, error) {
	query := `
		SELECT id, family, normalized_family, taxonomy_version, created_at, updated_at
		FROM job_families
		WHERE id = $1
	`

	var jobFamily models.JobFamily
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobFamily.ID,
		&jobFamily.Family,
		&jobFamily.NormalizedFamily,
		&jobFamily.TaxonomyVersion,
		&jobFamily.CreatedAt,
		&jobFamily.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobFamily, nil
}

// List returns all job families
func (r *JobFamilyRepository) List(ctx context.Context) ([]*models.JobFamily, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, family, normalized_family, taxonomy_version, created_at, updated_at
		FROM job_families
		ORDER BY family
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobFamilies []*models.JobFamily
	for rows.Next() {
		var jobFamily models.JobFamily
		if err := rows.Scan(
			&jobFamily.ID,
			&jobFamily.Family,
			&jobFamily.NormalizedFamily,
			&jobFamily.TaxonomyVersion,
			&jobFamily.CreatedAt,
			&jobFamily.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobFamilies = append(jobFamilies, &jobFamily)
	}

	return jobFamilies, rows.Err()
}

// Update updates an existing job family
func (r *JobFamilyRepository) Update(ctx context.Context, jobFamily *models.JobFamily) error {
	query := `
		UPDATE job_families
		SET family = $2,
		    normalized_family = $3,
		    taxonomy_version = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobFamily.ID,
		jobFamily.Family,
		jobFamily.NormalizedFamily,
		jobFamily.TaxonomyVersion,
		jobFamily.UpdatedAt,
	)

	return err
}

// Delete removes a job family by its ID
func (r *JobFamilyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_families
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}