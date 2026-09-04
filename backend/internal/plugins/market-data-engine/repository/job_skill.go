package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobSkillRepository handles storage and retrieval of job skills
type JobSkillRepository struct {
	db *pgxpool.Pool
}

// NewJobSkillRepository creates a new JobSkillRepository
func NewJobSkillRepository(db *pgxpool.Pool) *JobSkillRepository {
	return &JobSkillRepository{db: db}
}

// Insert adds a new job skill and returns its ID
func (r *JobSkillRepository) Insert(ctx context.Context, jobSkill *models.JobSkill) (uuid.UUID, error) {
	query := `
		INSERT INTO job_skills
			(id, job_posting_id, skill_id, skill_level, importance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobSkill.ID,
		jobSkill.JobPostingID,
		jobSkill.SkillID,
		jobSkill.SkillLevel,
		jobSkill.Importance,
		jobSkill.CreatedAt,
		jobSkill.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job skill by its ID
func (r *JobSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobSkill, error) {
	query := `
		SELECT id, job_posting_id, skill_id, skill_level, importance, created_at, updated_at
		FROM job_skills
		WHERE id = $1
	`

	var jobSkill models.JobSkill
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobSkill.ID,
		&jobSkill.JobPostingID,
		&jobSkill.SkillID,
		&jobSkill.SkillLevel,
		&jobSkill.Importance,
		&jobSkill.CreatedAt,
		&jobSkill.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobSkill, nil
}

// ListByJobPostingID returns all skills for a given job posting
func (r *JobSkillRepository) ListByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) ([]*models.JobSkill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, skill_id, skill_level, importance, created_at, updated_at
		FROM job_skills
		WHERE job_posting_id = $1
		ORDER BY skill_level DESC, importance DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSkills []*models.JobSkill
	for rows.Next() {
		var jobSkill models.JobSkill
		if err := rows.Scan(
			&jobSkill.ID,
			&jobSkill.JobPostingID,
			&jobSkill.SkillID,
			&jobSkill.SkillLevel,
			&jobSkill.Importance,
			&jobSkill.CreatedAt,
			&jobSkill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSkills = append(jobSkills, &jobSkill)
	}

	return jobSkills, rows.Err()
}

// ListBySkillID returns all job postings that require a given skill
func (r *JobSkillRepository) ListBySkillID(ctx context.Context, skillID uuid.UUID) ([]*models.JobSkill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, skill_id, skill_level, importance, created_at, updated_at
		FROM job_skills
		WHERE skill_id = $1
		ORDER BY job_posting_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSkills []*models.JobSkill
	for rows.Next() {
		var jobSkill models.JobSkill
		if err := rows.Scan(
			&jobSkill.ID,
			&jobSkill.JobPostingID,
			&jobSkill.SkillID,
			&jobSkill.SkillLevel,
			&jobSkill.Importance,
			&jobSkill.CreatedAt,
			&jobSkill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSkills = append(jobSkills, &jobSkill)
	}

	return jobSkills, rows.Err()
}

// Update updates an existing job skill
func (r *JobSkillRepository) Update(ctx context.Context, jobSkill *models.JobSkill) error {
	query := `
		UPDATE job_skills
		SET job_posting_id = $2,
		    skill_id = $3,
		    skill_level = $4,
		    importance = $5,
		    updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobSkill.ID,
		jobSkill.JobPostingID,
		jobSkill.SkillID,
		jobSkill.SkillLevel,
		jobSkill.Importance,
		jobSkill.UpdatedAt,
	)

	return err
}

// Delete removes a job skill by its ID
func (r *JobSkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_skills
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobPostingID removes all job skills for a given job posting
func (r *JobSkillRepository) DeleteByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) error {
	query := `
		DELETE FROM job_skills
		WHERE job_posting_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobPostingID)
	return err
}