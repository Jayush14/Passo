package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// SkillAliasRepository handles storage and retrieval of skill aliases
type SkillAliasRepository struct {
	db *pgxpool.Pool
}

// NewSkillAliasRepository creates a new SkillAliasRepository
func NewSkillAliasRepository(db *pgxpool.Pool) *SkillAliasRepository {
	return &SkillAliasRepository{db: db}
}

// Insert adds a new skill alias and returns its ID
func (r *SkillAliasRepository) Insert(ctx context.Context, alias *models.SkillAlias) (uuid.UUID, error) {
	query := `
		INSERT INTO skill_aliases
			(id, skill_id, alias, normalized_alias, source, confidence, status,
			 taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		alias.ID,
		alias.SkillID,
		alias.Alias,
		alias.NormalizedAlias,
		alias.Source,
		alias.Confidence,
		alias.Status,
		alias.TaxonomyVersion,
		alias.CreatedAt,
		alias.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a skill alias by its ID
func (r *SkillAliasRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SkillAlias, error) {
	query := `
		SELECT id, skill_id, alias, normalized_alias, source, confidence, status,
		       taxonomy_version, created_at, updated_at
		FROM skill_aliases
		WHERE id = $1
	`

	var alias models.SkillAlias
	err := r.db.QueryRow(ctx, query, id).Scan(
		&alias.ID,
		&alias.SkillID,
		&alias.Alias,
		&alias.NormalizedAlias,
		&alias.Source,
		&alias.Confidence,
		&alias.Status,
		&alias.TaxonomyVersion,
		&alias.CreatedAt,
		&alias.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alias, nil
}

// GetByNormalizedAlias retrieves a skill alias by its normalized alias
func (r *SkillAliasRepository) GetByNormalizedAlias(ctx context.Context, normalizedAlias string) (*models.SkillAlias, error) {
	query := `
		SELECT id, skill_id, alias, normalized_alias, source, confidence, status,
		       taxonomy_version, created_at, updated_at
		FROM skill_aliases
		WHERE normalized_alias = $1
	`

	var alias models.SkillAlias
	err := r.db.QueryRow(ctx, query, normalizedAlias).Scan(
		&alias.ID,
		&alias.SkillID,
		&alias.Alias,
		&alias.NormalizedAlias,
		&alias.Source,
		&alias.Confidence,
		&alias.Status,
		&alias.TaxonomyVersion,
		&alias.CreatedAt,
		&alias.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alias, nil
}

// ListBySkillID returns all aliases for a given skill
func (r *SkillAliasRepository) ListBySkillID(ctx context.Context, skillID uuid.UUID) ([]*models.SkillAlias, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, skill_id, alias, normalized_alias, source, confidence, status,
		       taxonomy_version, created_at, updated_at
		FROM skill_aliases
		WHERE skill_id = $1
		ORDER BY alias
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []*models.SkillAlias
	for rows.Next() {
		var alias models.SkillAlias
		if err := rows.Scan(
			&alias.ID,
			&alias.SkillID,
			&alias.Alias,
			&alias.NormalizedAlias,
			&alias.Source,
			&alias.Confidence,
			&alias.Status,
			&alias.TaxonomyVersion,
			&alias.CreatedAt,
			&alias.UpdatedAt,
		); err != nil {
			return nil, err
		}
		aliases = append(aliases, &alias)
	}

	return aliases, rows.Err()
}

// Update updates an existing skill alias
func (r *SkillAliasRepository) Update(ctx context.Context, alias *models.SkillAlias) error {
	query := `
		UPDATE skill_aliases
		SET skill_id = $2,
		    alias = $3,
		    normalized_alias = $4,
		    source = $5,
		    confidence = $6,
		    status = $7,
		    taxonomy_version = $8,
		    updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		alias.ID,
		alias.SkillID,
		alias.Alias,
		alias.NormalizedAlias,
		alias.Source,
		alias.Confidence,
		alias.Status,
		alias.TaxonomyVersion,
		alias.UpdatedAt,
	)

	return err
}

// Delete removes a skill alias by its ID
func (r *SkillAliasRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM skill_aliases
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}