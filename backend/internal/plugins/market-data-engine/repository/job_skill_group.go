package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobSkillGroupRepository handles storage and retrieval of job skill groups
type JobSkillGroupRepository struct {
	db *pgxpool.Pool
}

// NewJobSkillGroupRepository creates a new JobSkillGroupRepository
func NewJobSkillGroupRepository(db *pgxpool.Pool) *JobSkillGroupRepository {
	return &JobSkillGroupRepository{db: db}
}

// Insert adds a new job skill group and returns its ID
func (r *JobSkillGroupRepository) Insert(ctx context.Context, jobSkillGroup *models.JobSkillGroup) (uuid.UUID, error) {
	query := `
		INSERT INTO job_skill_groups
			(id, job_posting_id, group_name, skill_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobSkillGroup.ID,
		jobSkillGroup.JobPostingID,
		jobSkillGroup.GroupName,
		jobSkillGroup.SkillCount,
		jobSkillGroup.CreatedAt,
		jobSkillGroup.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job skill group by its ID
func (r *JobSkillGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobSkillGroup, error) {
	query := `
		SELECT id, job_posting_id, group_name, skill_count, created_at, updated_at
		FROM job_skill_groups
		WHERE id = $1
	`

	var jobSkillGroup models.JobSkillGroup
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobSkillGroup.ID,
		&jobSkillGroup.JobPostingID,
		&jobSkillGroup.GroupName,
		&jobSkillGroup.SkillCount,
		&jobSkillGroup.CreatedAt,
		&jobSkillGroup.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobSkillGroup, nil
}

// ListByJobPostingID returns all skill groups for a given job posting
func (r *JobSkillGroupRepository) ListByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) ([]*models.JobSkillGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, group_name, skill_count, created_at, updated_at
		FROM job_skill_groups
		WHERE job_posting_id = $1
		ORDER BY group_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSkillGroups []*models.JobSkillGroup
	for rows.Next() {
		var jobSkillGroup models.JobSkillGroup
		if err := rows.Scan(
			&jobSkillGroup.ID,
			&jobSkillGroup.JobPostingID,
			&jobSkillGroup.GroupName,
			&jobSkillGroup.SkillCount,
			&jobSkillGroup.CreatedAt,
			&jobSkillGroup.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSkillGroups = append(jobSkillGroups, &jobSkillGroup)
	}

	return jobSkillGroups, rows.Err()
}

// Update updates an existing job skill group
func (r *JobSkillGroupRepository) Update(ctx context.Context, jobSkillGroup *models.JobSkillGroup) error {
	query := `
		UPDATE job_skill_groups
		SET job_posting_id = $2,
		    group_name = $3,
		    skill_count = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobSkillGroup.ID,
		jobSkillGroup.JobPostingID,
		jobSkillGroup.GroupName,
		jobSkillGroup.SkillCount,
		jobSkillGroup.UpdatedAt,
	)

	return err
}

// Delete removes a job skill group by its ID
func (r *JobSkillGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_skill_groups
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobPostingID removes all job skill groups for a given job posting
func (r *JobSkillGroupRepository) DeleteByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) error {
	query := `
		DELETE FROM job_skill_groups
		WHERE job_posting_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobPostingID)
	return err
}