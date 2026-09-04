package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobTitleRepository handles storage and retrieval of job titles
type JobTitleRepository struct {
	db *pgxpool.Pool
}

// NewJobTitleRepository creates a new JobTitleRepository
func NewJobTitleRepository(db *pgxpool.Pool) *JobTitleRepository {
	return &JobTitleRepository{db: db}
}

// Insert adds a new job title and returns its ID
func (r *JobTitleRepository) Insert(ctx context.Context, jobTitle *models.JobTitle) (uuid.UUID, error) {
	query := `
		INSERT INTO job_titles
			(id, title, normalized_title, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobTitle.ID,
		jobTitle.Title,
		jobTitle.NormalizedTitle,
		jobTitle.TaxonomyVersion,
		jobTitle.CreatedAt,
		jobTitle.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job title by its ID
func (r *JobTitleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobTitle, error) {
	query := `
		SELECT id, title, normalized_title, taxonomy_version, created_at, updated_at
		FROM job_titles
		WHERE id = $1
	`

	var jobTitle models.JobTitle
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobTitle.ID,
		&jobTitle.Title,
		&jobTitle.NormalizedTitle,
		&jobTitle.TaxonomyVersion,
		&jobTitle.CreatedAt,
		&jobTitle.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobTitle, nil
}

// List returns all job titles
func (r *JobTitleRepository) List(ctx context.Context) ([]*models.JobTitle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, normalized_title, taxonomy_version, created_at, updated_at
		FROM job_titles
		ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobTitles []*models.JobTitle
	for rows.Next() {
		var jobTitle models.JobTitle
		if err := rows.Scan(
			&jobTitle.ID,
			&jobTitle.Title,
			&jobTitle.NormalizedTitle,
			&jobTitle.TaxonomyVersion,
			&jobTitle.CreatedAt,
			&jobTitle.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobTitles = append(jobTitles, &jobTitle)
	}

	return jobTitles, rows.Err()
}

// Update updates an existing job title
func (r *JobTitleRepository) Update(ctx context.Context, jobTitle *models.JobTitle) error {
	query := `
		UPDATE job_titles
		SET title = $2,
		    normalized_title = $3,
		    taxonomy_version = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobTitle.ID,
		jobTitle.Title,
		jobTitle.NormalizedTitle,
		jobTitle.TaxonomyVersion,
		jobTitle.UpdatedAt,
	)

	return err
}

// Delete removes a job title by its ID
func (r *JobTitleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_titles
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}