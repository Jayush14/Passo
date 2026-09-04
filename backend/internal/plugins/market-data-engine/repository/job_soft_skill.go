package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobSoftSkillRepository handles storage and retrieval of job soft skills
type JobSoftSkillRepository struct {
	db *pgxpool.Pool
}

// NewJobSoftSkillRepository creates a new JobSoftSkillRepository
func NewJobSoftSkillRepository(db *pgxpool.Pool) *JobSoftSkillRepository {
	return &JobSoftSkillRepository{db: db}
}

// Insert adds a new job soft skill and returns its ID
func (r *JobSoftSkillRepository) Insert(ctx context.Context, jobSoftSkill *models.JobSoftSkill) (uuid.UUID, error) {
	query := `
		INSERT INTO job_soft_skills
			(id, job_posting_id, soft_skill_id, proficiency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobSoftSkill.ID,
		jobSoftSkill.JobPostingID,
		jobSoftSkill.SoftSkillID,
		jobSoftSkill.Proficiency,
		jobSoftSkill.CreatedAt,
		jobSoftSkill.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job soft skill by its ID
func (r *JobSoftSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobSoftSkill, error) {
	query := `
		SELECT id, job_posting_id, soft_skill_id, proficiency, created_at, updated_at
		FROM job_soft_skills
		WHERE id = $1
	`

	var jobSoftSkill models.JobSoftSkill
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobSoftSkill.ID,
		&jobSoftSkill.JobPostingID,
		&jobSoftSkill.SoftSkillID,
		&jobSoftSkill.Proficiency,
		&jobSoftSkill.CreatedAt,
		&jobSoftSkill.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobSoftSkill, nil
}

// ListByJobPostingID returns all soft skills for a given job posting
func (r *JobSoftSkillRepository) ListByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) ([]*models.JobSoftSkill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, soft_skill_id, proficiency, created_at, updated_at
		FROM job_soft_skills
		WHERE job_posting_id = $1
		ORDER BY proficiency DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSoftSkills []*models.JobSoftSkill
	for rows.Next() {
		var jobSoftSkill models.JobSoftSkill
		if err := rows.Scan(
			&jobSoftSkill.ID,
			&jobSoftSkill.JobPostingID,
			&jobSoftSkill.SoftSkillID,
			&jobSoftSkill.Proficiency,
			&jobSoftSkill.CreatedAt,
			&jobSoftSkill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSoftSkills = append(jobSoftSkills, &jobSoftSkill)
	}

	return jobSoftSkills, rows.Err()
}

// ListBySoftSkillID returns all job postings that require a given soft skill
func (r *JobSoftSkillRepository) ListBySoftSkillID(ctx context.Context, softSkillID uuid.UUID) ([]*models.JobSoftSkill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, soft_skill_id, proficiency, created_at, updated_at
		FROM job_soft_skills
		WHERE soft_skill_id = $1
		ORDER BY job_posting_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSoftSkills []*models.JobSoftSkill
	for rows.Next() {
		var jobSoftSkill models.JobSoftSkill
		if err := rows.Scan(
			&jobSoftSkill.ID,
			&jobSoftSkill.JobPostingID,
			&jobSoftSkill.SoftSkillID,
			&jobSoftSkill.Proficiency,
			&jobSoftSkill.CreatedAt,
			&jobSoftSkill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSoftSkills = append(jobSoftSkills, &jobSoftSkill)
	}

	return jobSoftSkills, rows.Err()
}

// Update updates an existing job soft skill
func (r *JobSoftSkillRepository) Update(ctx context.Context, jobSoftSkill *models.JobSoftSkill) error {
	query := `
		UPDATE job_soft_skills
		SET job_posting_id = $2,
		    soft_skill_id = $3,
		    proficiency = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobSoftSkill.ID,
		jobSoftSkill.JobPostingID,
		jobSoftSkill.SoftSkillID,
		jobSoftSkill.Proficiency,
		jobSoftSkill.UpdatedAt,
	)

	return err
}

// Delete removes a job soft skill by its ID
func (r *JobSoftSkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_soft_skills
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobPostingID removes all job soft skills for a given job posting
func (r *JobSoftSkillRepository) DeleteByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) error {
	query := `
		DELETE FROM job_soft_skills
		WHERE job_posting_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobPostingID)
	return err
}